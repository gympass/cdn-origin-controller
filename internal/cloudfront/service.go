// Copyright (c) 2021 GPBR Participacoes LTDA.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package cloudfront

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
	"github.com/Gympass/cdn-origin-controller/internal/certificate"
	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"
	"github.com/Gympass/cdn-origin-controller/internal/route53"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	reasonFailed  = "FailedToReconcile"
	reasonSuccess = "SuccessfullyReconciled"
)

// Service handles operations involving CloudFront
type Service struct {
	client.Client

	Config      config.Config
	Recorder    record.EventRecorder
	AliasRepo   route53.AliasRepository
	DistRepo    DistributionRepository
	Fetcher     k8s.IngressFetcher
	CertService certificate.Service
}

// Reconcile an Ingress resource of any version
func (s *Service) Reconcile(ctx context.Context, ing *networkingv1.Ingress, class k8s.CDNClass) error {
	if err := s.validateIngress(ing); err != nil {
		return s.handleFailure(fmt.Errorf("validating Ingress: %v", err), ing)
	}

	reconciling, err := k8s.NewCDNIngressFromV1(ctx, ing, class)
	if err != nil {
		return s.handleFailure(err, ing)
	}

	log, _ := logr.FromContext(ctx)

	if k8s.HasFinalizer(ing) && !k8s.HasGroupAnnotation(ing) {
		err := errors.New("ingress has no group annotation but has finalizer, can't continue without a group")
		log.Error(err, "Faced invalid Ingress, removing finalizer. State may be inconsistent but should eventually self-heal.")
		return s.reconcileFinalizer(ing, false)
	}

	desiredIngresses, desiredDist, err := s.desiredState(ctx, reconciling)
	if err != nil {
		return s.handleFailure(fmt.Errorf("computing desired state: %v", err), ing)
	}

	if err := s.validateCreation(desiredDist, ing); err != nil {
		return s.handleFailure(err, ing)
	}

	cdnStatus, err := s.fetchOrGenerateCDNStatus(desiredIngresses, desiredDist)
	if err != nil {
		return s.handleFailure(fmt.Errorf("validating creation: %v", err), ing)
	}

	errs := &multierror.Error{}

	existingDist, err := s.syncDist(ctx, desiredDist, cdnStatus, ing)
	errs = multierror.Append(errs, err)

	if reconciling.Class.CreateAlias {
		err := s.syncAliases(cdnStatus, existingDist, reconciling.Class)
		errs = multierror.Append(errs, err)
	}

	shouldHaveFinalizer := errs.Len() > 0 || !k8s.IsBeingRemovedFromDesiredState(ing)
	if err := s.reconcileFinalizer(ing, shouldHaveFinalizer); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("reconciling finalizer for ingress (%s/%s): %v", ing.GetNamespace(), ing.GetName(), err))
	}

	if errs.Len() == 0 && k8s.IsBeingRemovedFromDesiredState(ing) {
		cdnStatus.RemoveIngressRef(ing)
	}

	if errs.Len() == 0 && desiredDist.IsEmpty() {
		errs = multierror.Append(errs, s.deleteCDNStatus(ctx, cdnStatus))
	} else {
		errs = multierror.Append(errs, s.upsertCDNStatus(ctx, cdnStatus))
	}

	return s.handleResult(ing, cdnStatus, errs)
}

func (s *Service) validateCreation(desiredDist Distribution, ing *networkingv1.Ingress) error {
	if desiredDist.Exists() || desiredDist.IsEmpty() || ing.DeletionTimestamp != nil {
		return nil
	}

	if !s.Config.IsCreationAllowed(ing) {
		return errors.New("creation of new CloudFront distributions is blocked")
	}

	return nil
}

func (s *Service) validateIngress(ing *networkingv1.Ingress) error {
	if df := k8s.UsedDeprecatedFields(ing); len(df) > 0 {
		s.Recorder.Eventf(
			ing,
			corev1.EventTypeWarning,
			"UsingDeprecatedFields",
			"Using deprecated fields/annotations: %v", df)
	}

	return k8s.ValidateIngressFunctionAssociations(ing)
}

func (s *Service) desiredState(ctx context.Context, reconciling k8s.CDNIngress) ([]k8s.CDNIngress, Distribution, error) {
	desiredIngresses, err := s.desiredIngresses(ctx, reconciling)
	if err != nil {
		return nil, Distribution{}, err
	}

	sharedParams, err := k8s.NewSharedIngressParams(desiredIngresses)
	if err != nil {
		return nil, Distribution{}, fmt.Errorf("shared ingress params: %v", err)
	}

	existingDistARN, err := s.DistRepo.ARNByGroup(reconciling.Group)
	if err != nil && !errors.Is(err, ErrDistNotFound) {
		return nil, Distribution{}, fmt.Errorf("fetching existing CloudFront ID based on group (%s): %v", reconciling.Group, err)
	}

	desiredDist, err := s.newDistribution(desiredIngresses, reconciling.Group, sharedParams, existingDistARN)
	if err != nil {
		return nil, Distribution{}, fmt.Errorf("building desired distribution: %v", err)
	}

	return desiredIngresses, desiredDist, nil
}

func (s *Service) desiredIngresses(ctx context.Context, reconciling k8s.CDNIngress) ([]k8s.CDNIngress, error) {
	desiredIngresses, err := s.Fetcher.FetchBy(ctx, reconciling.Class, s.isPartOfDesiredState(reconciling))
	if err != nil {
		return nil, fmt.Errorf("listing active Ingresses that belong to group %s: %v", reconciling.Group, err)
	}
	return desiredIngresses, nil
}

func (s *Service) isPartOfDesiredState(reconciling k8s.CDNIngress) func(k8s.CDNIngress) bool {
	return func(ing k8s.CDNIngress) bool {
		isPartOfGroup := ing.Group == reconciling.Group
		hasBeenProvisioned := len(ing.OriginHost) > 0
		return !ing.IsBeingRemoved && isPartOfGroup && hasBeenProvisioned
	}
}

func (s *Service) fetchOrGenerateCDNStatus(desiredIngs []k8s.CDNIngress, dist Distribution) (*v1alpha1.CDNStatus, error) {
	status := &v1alpha1.CDNStatus{}
	key := client.ObjectKey{Name: dist.Group}

	err := s.Client.Get(context.Background(), key, status)
	if k8serrors.IsNotFound(err) {
		return newCDNStatus(desiredIngs, dist), nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching CDNStatus: %v", err)
	}

	return status, nil
}

func newCDNStatus(ings []k8s.CDNIngress, dist Distribution) *v1alpha1.CDNStatus {
	status := &v1alpha1.CDNStatus{
		ObjectMeta: metav1.ObjectMeta{
			Name: dist.Group,
		},
	}

	for _, ing := range ings {
		status.SetIngressRef(false, ing)
	}

	return status
}

func (s *Service) newDistribution(ingresses []k8s.CDNIngress, group string, shared k8s.SharedIngressParams, distARN string) (Distribution, error) {
	b := NewDistributionBuilder(
		group,
		s.Config,
	)
	var err error
	var cert certificate.Certificate
	if s.Config.TLSIsEnabled() {
		cert, err = s.discoverCert(ingresses)
		if err != nil {
			return Distribution{}, fmt.Errorf("discovering TLS cert: %v", err)
		}
		b = b.WithTLS(cert.ARN(), s.Config.CloudFrontSecurityPolicy)
	}

	for _, ing := range ingresses {
		b = b.WithOrigin(newOrigin(ing, s.Config, shared))
		b = b.WithAlternateDomains(ing.AlternateDomainNames)
		b = b.AppendTags(ing.Tags)
	}

	if s.Config.CloudFrontEnableIPV6 {
		b = b.WithIPv6()
	}

	if s.Config.CloudFrontEnableLogging && len(s.Config.CloudFrontS3BucketLog) > 0 {
		b = b.WithLogging(s.Config.CloudFrontS3BucketLog, s.s3Prefix(group))
	}

	if len(s.Config.CloudFrontCustomTags) > 0 {
		b = b.AppendTags(s.Config.CloudFrontCustomTags)
	}

	if len(shared.WebACLARN) > 0 {
		b = b.WithWebACL(shared.WebACLARN)
	}

	if len(distARN) > 0 {
		b = b.WithARN(distARN)
	}

	return b.Build()
}

// discoverCert returns the first found ACM Certificate that matches any Alternate Domain Name of the input Ingresses
func (s *Service) discoverCert(ingresses []k8s.CDNIngress) (certificate.Certificate, error) {
	var alternateDomains []string
	for _, ing := range ingresses {
		alternateDomains = append(alternateDomains, ing.AlternateDomainNames...)
	}

	cert, err := s.CertService.DiscoverByHost(alternateDomains)
	if err != nil {
		return certificate.Certificate{}, fmt.Errorf("%v: %v", alternateDomains, err)
	}

	return cert, nil
}

func (s *Service) s3Prefix(group string) string {
	if len(s.Config.CloudFrontS3BucketLogPrefix) == 0 {
		return group
	}
	return fmt.Sprintf("%s/%s", s.Config.CloudFrontS3BucketLogPrefix, group)
}

func (s *Service) syncDist(ctx context.Context, desiredDist Distribution, cdnStatus *v1alpha1.CDNStatus, ing client.Object) (Distribution, error) {
	if desiredDist.IsEmpty() {
		return desiredDist, s.deleteDistribution(ctx, desiredDist)
	}
	return s.upsertDistribution(ctx, desiredDist, cdnStatus, ing)
}

func (s *Service) upsertDistribution(ctx context.Context, dist Distribution, status *v1alpha1.CDNStatus, ing client.Object) (Distribution, error) {
	var err error
	var existingDist Distribution

	if dist.Exists() {
		existingDist, err = s.updateDistribution(ctx, dist)
	} else {
		existingDist, err = s.createDistribution(ctx, dist)
	}

	status.SetIngressRef(err == nil, ing)
	if err == nil {
		status.SetInfo(existingDist.ID, existingDist.ARN, existingDist.Address)
		status.SetAliases(existingDist.AlternateDomains)
	}

	return existingDist, err
}

func (s *Service) updateDistribution(ctx context.Context, dist Distribution) (Distribution, error) {
	log, _ := logr.FromContext(ctx)
	log.V(1).Info("Updating existing Distribution.", "distribution", dist)

	existingDist, err := s.DistRepo.Sync(dist)
	if err != nil {
		return Distribution{}, fmt.Errorf("updating Distribution: %v", err)
	}

	return existingDist, nil
}

func (s *Service) createDistribution(ctx context.Context, dist Distribution) (Distribution, error) {
	log, _ := logr.FromContext(ctx)
	log.V(1).Info("Creating Distribution.", "distribution", dist)

	existingDist, err := s.DistRepo.Create(dist)
	if err != nil {
		return Distribution{}, fmt.Errorf("creating Distribution: %v", err)
	}

	return existingDist, nil
}

func (s *Service) deleteDistribution(ctx context.Context, dist Distribution) error {
	if !dist.Exists() {
		return nil
	}

	log, _ := logr.FromContext(ctx)
	if !s.Config.DeletionEnabled {
		log.V(1).Info("In a deletion operation, but configured not to delete Distributions. Will not delete.")
		return nil
	}

	log.V(1).Info("Disabling and deleting distribution on AWS, this may take a few minutes.")
	return s.DistRepo.Delete(dist)
}

func (s *Service) upsertCDNStatus(ctx context.Context, status *v1alpha1.CDNStatus) error {
	if !status.Exists() {
		// create does not touch the .status subresource, so we need to create, then update the status
		if err := s.Create(ctx, status); err != nil {
			return err
		}
	}
	return s.Status().Update(ctx, status)
}

func (s *Service) deleteCDNStatus(ctx context.Context, cdnStatus *v1alpha1.CDNStatus) error {
	if err := s.Delete(ctx, cdnStatus); err != nil && !k8serrors.IsNotFound(err) {
		log, _ := logr.FromContext(ctx)
		log.V(1).Error(err, "Could not delete CDNStatus resource", "cdnStatus", cdnStatus)
		return err
	}
	return nil
}

func (s *Service) syncAliases(cdnStatus *v1alpha1.CDNStatus, dist Distribution, class k8s.CDNClass) error {
	upserting, deleting := s.newAliases(dist, cdnStatus, class)

	errUpsert := s.AliasRepo.Upsert(upserting)
	if errUpsert == nil {
		cdnStatus.UpsertDNSRecords(upserting.Domains())
	}
	errDelete := s.AliasRepo.Delete(deleting)
	if errDelete == nil {
		cdnStatus.RemoveDNSRecords(deleting.Domains())
	}
	var result *multierror.Error
	if errUpsert != nil || errDelete != nil {
		cdnStatus.SetDNSSync(false)
		result = multierror.Append(result, errUpsert, errDelete)
	}

	return result.ErrorOrNil()
}

func (s *Service) newAliases(dist Distribution, status *v1alpha1.CDNStatus, class k8s.CDNClass) (route53.Aliases, route53.Aliases) {
	var deleting []string
	if status.Status.DNS != nil {
		desiredDomains := route53.NormalizeDomains(dist.AlternateDomains)
		existingDomains := status.Status.DNS.Records
		deleting = getDeletions(desiredDomains, existingDomains)
	}

	if !s.Config.DeletionEnabled {
		deleting = []string{}
	}

	toUpsert := route53.NewAliases(dist.Address, class.HostedZoneID, class.TXTOwnerValue, dist.AlternateDomains, dist.IPv6Enabled)
	toDelete := route53.NewAliases(dist.Address, class.HostedZoneID, class.TXTOwnerValue, deleting, dist.IPv6Enabled)

	return toUpsert, toDelete
}

func getDeletions(desiredDomains, currentDomains []string) []string {
	var toDelete []string
	for _, currentDomain := range currentDomains {
		if !strhelper.Contains(desiredDomains, currentDomain) {
			toDelete = append(toDelete, currentDomain)
		}
	}
	return toDelete
}

func (s *Service) reconcileFinalizer(obj client.Object, shouldHaveFinalizer bool) error {
	if shouldHaveFinalizer {
		k8s.AddFinalizer(obj)
	} else {
		k8s.RemoveFinalizer(obj)
	}
	return s.Client.Update(context.Background(), obj)
}

func (s *Service) handleResult(obj client.Object, cdnStatus *v1alpha1.CDNStatus, errs *multierror.Error) error {
	if errs.Len() > 0 {
		return s.handleFailureWithStatus(errs, obj, cdnStatus)
	}
	return s.handleSuccess(obj, cdnStatus)
}

func (s *Service) handleFailure(err error, ingress client.Object) error {
	s.recordFailureOnIngress(err, ingress)
	return err
}

func (s *Service) handleFailureWithStatus(err error, ingress client.Object, status *v1alpha1.CDNStatus) error {
	msg := s.recordFailureOnIngress(err, ingress)

	ingRef := v1alpha1.NewIngressRef(ingress.GetNamespace(), ingress.GetName())
	msg = fmt.Sprintf("%s: %s", ingRef, msg)
	s.Recorder.Event(status, corev1.EventTypeWarning, reasonFailed, msg)

	return err
}

func (s *Service) recordFailureOnIngress(err error, ingress client.Object) string {
	msg := "Unable to reconcile CDN: " + err.Error()
	s.Recorder.Event(ingress, corev1.EventTypeWarning, reasonFailed, msg)
	return msg
}

func (s *Service) handleSuccess(ingress client.Object, status *v1alpha1.CDNStatus) error {
	msg := "Successfully reconciled CDN"
	s.Recorder.Event(ingress, corev1.EventTypeNormal, reasonSuccess, msg)

	status.SetDNSSync(true)

	ingRef := v1alpha1.NewIngressRef(ingress.GetNamespace(), ingress.GetName())
	msg = fmt.Sprintf("%s: %s", ingRef, msg)
	s.Recorder.Event(status, corev1.EventTypeNormal, reasonSuccess, msg)

	return nil
}
