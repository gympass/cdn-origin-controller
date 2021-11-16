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

package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/hashicorp/go-multierror"

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/route53"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	cdnGroupAnnotation               = "cdn-origin-controller.gympass.com/cdn.group"
	cfViewerFnAnnotation             = "cdn-origin-controller.gympass.com/cf.viewer-function-arn"
	cfOrigRespTimeoutAnnotation      = "cdn-origin-controller.gympass.com/cf.origin-response-timeout"
	cfAlternateDomainNamesAnnotation = "cdn-origin-controller.gympass.com/cf.alternate-domain-names"
)

const (
	reasonFailed  = "FailedToReconcile"
	reasonSuccess = "SuccessfullyReconciled"
)

var errNoAnnotation = errors.New(cdnGroupAnnotation + " annotation not present")

type boundIngressesFunc func(v1alpha1.CDNStatus) ([]ingressParams, error)

// IngressReconciler reconciles Ingress resources of any version
type IngressReconciler struct {
	client.Client

	BoundIngressParamsFn boundIngressesFunc
	Config               config.Config
	Recorder             record.EventRecorder
	AliasRepo            route53.AliasRepository
	DistRepo             cloudfront.DistributionRepository

	log logr.Logger
}

// Reconcile an Ingress resource of any version
func (r *IngressReconciler) Reconcile(reconciling ingressParams, obj client.Object) error {
	dist := newDistribution(newOrigin(reconciling), reconciling, r.Config)

	var errs *multierror.Error
	var reconciliationErr error
	nsName := types.NamespacedName{Name: reconciling.group}
	cdnStatus := &v1alpha1.CDNStatus{}
	err := r.Get(context.Background(), nsName, cdnStatus)

	shouldCreate := k8serrors.IsNotFound(err)
	shouldSync := err == nil

	switch {
	case shouldCreate:
		dist, reconciliationErr = r.createDistribution(dist, reconciling.group)
		errs = multierror.Append(errs, reconciliationErr)

		cdnStatus, err = r.createCDNStatus(dist, reconciling.group)
		if err != nil {
			errs = multierror.Append(errs, err)
			return fmt.Errorf("creating CDNStatus: %v", errs.ErrorOrNil())
		}
	case shouldSync:
		dist, reconciliationErr = r.syncDistribution(cdnStatus, obj, dist)
		errs = multierror.Append(errs, reconciliationErr)
	default:
		return fmt.Errorf("fetching CDN status: %v", err)
	}

	inSync := true
	if errs.Len() > 0 {
		inSync = false
	}
	cdnStatus.SetIngressRef(inSync, obj)
	cdnStatus.SetAliases(dist.AlternateDomains)

	if r.Config.CloudFrontRoute53CreateAlias {
		err := r.syncAliases(cdnStatus, dist)
		errs = multierror.Append(errs, err)
	}

	return r.handleResult(obj, cdnStatus, errs)
}

func (r *IngressReconciler) handleResult(obj client.Object, cdnStatus *v1alpha1.CDNStatus, errs *multierror.Error) error {
	if err := r.updateCDNStatus(cdnStatus); err != nil {
		errs = multierror.Append(errs, err)
	}

	if errs.Len() > 0 {
		return r.handleFailure(errs, obj, cdnStatus)
	}

	return r.handleSuccess(obj, cdnStatus)
}

func (r *IngressReconciler) syncAliases(cdnStatus *v1alpha1.CDNStatus, dist cloudfront.Distribution) error {
	if cdnStatus.Status.DNS == nil {
		cdnStatus.Status.DNS = &v1alpha1.DNSStatus{Synced: true}
	}

	upserting, deleting := r.newAliases(dist, cdnStatus)

	errUpsert := r.AliasRepo.Upsert(upserting)
	if errUpsert == nil {
		cdnStatus.UpsertDNSRecords(upserting.Domains())
	}
	errDelete := r.AliasRepo.Delete(deleting)
	if errUpsert == nil {
		cdnStatus.RemoveDNSRecords(deleting.Domains())
	}
	var result *multierror.Error
	if errUpsert != nil || errDelete != nil {
		cdnStatus.Status.DNS.Synced = false
		result = multierror.Append(result, errUpsert, errDelete)
	}

	return result.ErrorOrNil()
}

func (r *IngressReconciler) syncDistribution(cdnStatus *v1alpha1.CDNStatus, obj client.Object, dist cloudfront.Distribution) (cloudfront.Distribution, error) {
	r.log.V(1).Info("CDNStatus resource found.", "cdnStatusName", cdnStatus.Name)
	boundIngressesParams, err := r.BoundIngressParamsFn(filterIngressRef(cdnStatus, obj))
	if err != nil {
		return dist, err
	}
	for _, ip := range boundIngressesParams {
		dist.AddOrigin(newOrigin(ip))
		dist.AddAlternateDomains(ip.alternateDomainNames)
	}

	dist.ID = cdnStatus.Status.ID
	dist.ARN = cdnStatus.Status.ARN
	dist.Address = cdnStatus.Status.Address
	r.log.V(1).Info("Built desired distribution.", "distribution", dist)

	return dist, r.DistRepo.Sync(dist)
}

func (r *IngressReconciler) createDistribution(dist cloudfront.Distribution, group string) (cloudfront.Distribution, error) {
	r.log.V(1).Info("Built desired distribution.", "distribution", dist)
	r.log.V(1).Info("CDNStatus resource not found, creating.", "cdnStatusName", group)
	modifiedDist, err := r.DistRepo.Create(dist)
	if err != nil {
		return dist, err
	}
	r.log.V(1).Info("Distribution created.", "distribution", dist)
	return modifiedDist, err
}

func (r *IngressReconciler) newAliases(dist cloudfront.Distribution, status *v1alpha1.CDNStatus) (toUpsert route53.Aliases, toDelete route53.Aliases) {
	var deleting []string
	if status.Status.DNS != nil {
		deleting = getDeletions(dist.AlternateDomains, status.Status.DNS.Records)
	}

	return route53.NewAliases(dist.Address, dist.AlternateDomains, dist.IPv6Enabled), route53.NewAliases(dist.Address, deleting, dist.IPv6Enabled)
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

func (r *IngressReconciler) createCDNStatus(dist cloudfront.Distribution, group string) (*v1alpha1.CDNStatus, error) {
	cdnStatus := &v1alpha1.CDNStatus{
		ObjectMeta: metav1.ObjectMeta{
			Name: group,
		},
		Status: v1alpha1.CDNStatusStatus{
			ID:      dist.ID,
			ARN:     dist.ARN,
			Aliases: dist.AlternateDomains,
			Address: dist.Address,
		},
	}

	err := r.Create(context.Background(), cdnStatus)
	if err != nil {
		return nil, err
	}
	return cdnStatus, nil
}

func (r *IngressReconciler) updateCDNStatus(status *v1alpha1.CDNStatus) error {
	if err := r.Status().Update(context.Background(), status); err != nil {
		r.log.Error(err, "Could not persist CDNStatus resource", "cdnStatus", status)
		return err
	}
	return nil
}

func (r *IngressReconciler) handleFailure(err error, ingress client.Object, status *v1alpha1.CDNStatus) error {
	msg := "Unable to reconcile CDN: " + err.Error()
	r.Recorder.Event(ingress, corev1.EventTypeWarning, reasonFailed, msg)

	ingRef := v1alpha1.NewIngressRef(ingress.GetNamespace(), ingress.GetName())
	msg = fmt.Sprintf("%s: %s", ingRef, msg)
	r.Recorder.Event(status, corev1.EventTypeWarning, reasonFailed, msg)

	return err
}

func (r *IngressReconciler) handleSuccess(ingress client.Object, status *v1alpha1.CDNStatus) error {
	msg := "Successfully reconciled CDN"
	r.Recorder.Event(ingress, corev1.EventTypeNormal, reasonSuccess, msg)

	ingRef := v1alpha1.NewIngressRef(ingress.GetNamespace(), ingress.GetName())
	msg = fmt.Sprintf("%s: %s", ingRef, msg)
	r.Recorder.Event(status, corev1.EventTypeNormal, reasonSuccess, msg)

	return nil
}

func filterIngressRef(status *v1alpha1.CDNStatus, obj client.Object) v1alpha1.CDNStatus {
	statusCopy := status.DeepCopy()
	for ref := range status.Status.Ingresses {
		if ref.GetName() == obj.GetName() && ref.GetNamespace() == obj.GetNamespace() {
			delete(statusCopy.Status.Ingresses, ref)
		}
	}
	return *statusCopy
}
