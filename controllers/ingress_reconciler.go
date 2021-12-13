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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/hashicorp/go-multierror"

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/route53"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	cdnFinalizer                     = "cdn-origin-controller.gympass.com/finalizer"
	cdnGroupAnnotation               = "cdn-origin-controller.gympass.com/cdn.group"
	cdnClassAnnotation               = "cdn-origin-controller.gympass.com/cdn.class"
	cfUserOriginsAnnotation          = "cdn-origin-controller.gympass.com/cf.user-origins"
	cfViewerFnAnnotation             = "cdn-origin-controller.gympass.com/cf.viewer-function-arn"
	cfOrigReqPolicyAnnotation        = "cdn-origin-controller.gympass.com/cf.origin-request-policy"
	cfOrigRespTimeoutAnnotation      = "cdn-origin-controller.gympass.com/cf.origin-response-timeout"
	cfAlternateDomainNamesAnnotation = "cdn-origin-controller.gympass.com/cf.alternate-domain-names"
)

const (
	reasonFailed  = "FailedToReconcile"
	reasonSuccess = "SuccessfullyReconciled"
)

var errNoCDNStatusForIng = errors.New("could not find a CDNStatus that referenced the ingress")

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
	var ingresses []ingressParams
	errs := &multierror.Error{}

	isRemovingReconciling := isRemoving(obj)
	if !isRemovingReconciling {
		ingresses = append(ingresses, reconciling)

		userOriginParams, err := r.ingressParamsForUserOrigins(reconciling.group, obj)
		errs = multierror.Append(errs, err)
		ingresses = append(ingresses, userOriginParams...)
	}

	cdnStatus, fetchStatusErr := r.fetchCDNStatus(obj)
	if errors.Is(fetchStatusErr, errNoCDNStatusForIng) {
		r.log.Error(fmt.Errorf("fetching CDNStatus: %v", fetchStatusErr), "CDNStatus not found for Ingress which has no group annotation but has finalizer. Removing finalizer. State may be inconsistent.")
		shouldHaveFinalizer := false
		return r.reconcileFinalizer(obj, shouldHaveFinalizer)
	}

	foundStatus := fetchStatusErr == nil
	if foundStatus {
		r.log.V(1).Info("CDNStatus resource found.", "cdnStatusName", cdnStatus.Name)
		boundIngresses, err := r.BoundIngressParamsFn(filterIngressRef(cdnStatus, obj))
		if err != nil {
			return err
		}
		ingresses = append(ingresses, boundIngresses...)

		if isRemovingReconciling {
			cdnStatus.RemoveIngressRef(obj)
		}
	}

	group := cdnStatus.Name
	if !foundStatus {
		group = reconciling.group
	}

	var distErr error
	var dist cloudfront.Distribution
	distBuilder := newDistributionBuilder(ingresses, group, r.Config)

	shouldCreateDist := k8serrors.IsNotFound(fetchStatusErr)
	shouldDeleteDist := len(ingresses) == 0
	shouldSyncDist := foundStatus && !shouldDeleteDist
	switch {
	case shouldCreateDist:
		dist, distErr = r.createDistribution(distBuilder, group)
		if distErr != nil {
			return fmt.Errorf("creating distribution: %v", distErr)
		}

		cdnStatus, distErr = r.createCDNStatus(dist, reconciling.group)
		if distErr != nil {
			return fmt.Errorf("creating CDNStatus: %v", distErr)
		}
	case shouldSyncDist:
		dist, distErr = r.syncDistribution(cdnStatus, distBuilder)
		errs = multierror.Append(errs, distErr)
	case shouldDeleteDist:
		dist, distErr = r.deleteDistribution(cdnStatus, distBuilder)
		errs = multierror.Append(errs, distErr)
	default:
		return fmt.Errorf("fetching CDN status: %v", fetchStatusErr)
	}

	if !isRemovingReconciling {
		inSync := true
		if errs.Len() > 0 {
			inSync = false
		}
		cdnStatus.SetIngressRef(inSync, obj)
	}
	cdnStatus.SetAliases(dist.AlternateDomains)

	var aliasesErr error
	if r.Config.CloudFrontRoute53CreateAlias {
		aliasesErr = r.syncAliases(cdnStatus, dist)
		errs = multierror.Append(errs, aliasesErr)
	}

	reconciledSomething := distErr == nil || aliasesErr == nil
	shouldHaveFinalizer := hasFinalizer(obj) || reconciledSomething
	if isRemovingReconciling {
		shouldHaveFinalizer = errs.Len() > 0
	}
	return r.handleResult(obj, cdnStatus, shouldHaveFinalizer, errs)
}

func (r *IngressReconciler) ingressParamsForUserOrigins(group string, obj client.Object) ([]ingressParams, error) {
	userOriginsMarkup, ok := obj.GetAnnotations()[cfUserOriginsAnnotation]
	if !ok {
		return nil, nil
	}
	r.log.V(1).Info("Found user origins annotation.", "value", userOriginsMarkup)

	origins, err := userOriginsFromYAML([]byte(userOriginsMarkup))
	if err != nil {
		return nil, fmt.Errorf("parsing user origins data from the %s annotation: %v", cfUserOriginsAnnotation, err)
	}
	r.log.V(1).Info("Parsed user origins annotation.", "origins", origins)

	var result []ingressParams
	for _, o := range origins {
		ip := ingressParams{
			destinationHost:   o.Host,
			group:             group,
			paths:             o.paths(),
			viewerFnARN:       o.ViewerFunctionARN,
			originReqPolicy:   o.RequestPolicy,
			originRespTimeout: o.ResponseTimeout,
		}
		result = append(result, ip)
		r.log.V(1).Info("Added user origin to desired state.", "parameters", ip)
	}

	return result, nil
}

func (r *IngressReconciler) fetchCDNStatus(ing client.Object) (*v1alpha1.CDNStatus, error) {
	if isRemoving(ing) && !hasGroupAnnotation(ing) {
		return r.discoverCDNStatusForIngress(ing)
	}

	nsName := types.NamespacedName{Name: ing.GetAnnotations()[cdnGroupAnnotation]}
	cdnStatus := &v1alpha1.CDNStatus{}
	fetchStatusErr := r.Get(context.Background(), nsName, cdnStatus)
	return cdnStatus, fetchStatusErr
}

func (r *IngressReconciler) discoverCDNStatusForIngress(ing client.Object) (*v1alpha1.CDNStatus, error) {
	statusList := &v1alpha1.CDNStatusList{}
	if err := r.Client.List(context.Background(), statusList); err != nil {
		return nil, fmt.Errorf("listing CDNStatus resources: %v", err)
	}

	for _, s := range statusList.Items {
		if s.HasIngressRef(ing) {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("trying to match Ingress (%s/%s): %w", ing.GetNamespace(), ing.GetName(), errNoCDNStatusForIng)
}

func (r *IngressReconciler) handleResult(obj client.Object, cdnStatus *v1alpha1.CDNStatus, shouldHaveFinalizer bool, errs *multierror.Error) error {
	err := r.reconcileStatus(cdnStatus)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("reconciling CDNStatus: %v", err))
	}
	err = r.reconcileFinalizer(obj, shouldHaveFinalizer)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("reconciling finalizer: %v", err))
	}

	if errs.Len() > 0 {
		return r.handleFailure(errs, obj, cdnStatus)
	}
	return r.handleSuccess(obj, cdnStatus)
}

func (r *IngressReconciler) reconcileStatus(cdnStatus *v1alpha1.CDNStatus) error {
	if len(cdnStatus.Status.Ingresses) == 0 {
		if err := r.Delete(context.Background(), cdnStatus); err != nil {
			r.log.Error(err, "Could not delete CDNStatus resource", "cdnStatus", cdnStatus)
			return err
		}
	}
	if err := r.updateCDNStatus(cdnStatus); err != nil {
		r.log.Error(err, "Could not persist CDNStatus resource", "cdnStatus", cdnStatus)
		return err
	}
	return nil
}

func (r *IngressReconciler) reconcileFinalizer(obj client.Object, shouldHaveFinalizer bool) error {
	if shouldHaveFinalizer {
		controllerutil.AddFinalizer(obj, cdnFinalizer)
	} else {
		controllerutil.RemoveFinalizer(obj, cdnFinalizer)
	}
	return r.Client.Update(context.Background(), obj)
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

func (r *IngressReconciler) createDistribution(builder cloudfront.DistributionBuilder, group string) (cloudfront.Distribution, error) {
	dist, err := r.build(builder)
	if err != nil {
		return cloudfront.Distribution{}, fmt.Errorf("building desired distribution: %v", err)
	}

	r.log.V(1).Info("CDNStatus resource not found, creating.", "cdnStatusName", group)
	modifiedDist, err := r.DistRepo.Create(dist)
	if err != nil {
		return dist, err
	}
	r.log.V(1).Info("Distribution created.", "distribution", dist)
	return modifiedDist, err
}

func (r *IngressReconciler) syncDistribution(cdnStatus *v1alpha1.CDNStatus, builder cloudfront.DistributionBuilder) (cloudfront.Distribution, error) {
	dist, err := r.build(builder.WithInfo(cdnStatus.Status.ID, cdnStatus.Status.ARN, cdnStatus.Status.Address))
	if err != nil {
		return cloudfront.Distribution{}, fmt.Errorf("building desired distribution: %v", err)
	}
	return dist, r.DistRepo.Sync(dist)
}

func (r *IngressReconciler) deleteDistribution(cdnStatus *v1alpha1.CDNStatus, builder cloudfront.DistributionBuilder) (cloudfront.Distribution, error) {
	dist, err := r.build(builder.WithInfo(cdnStatus.Status.ID, cdnStatus.Status.ARN, cdnStatus.Status.Address))
	if err != nil {
		return cloudfront.Distribution{}, fmt.Errorf("building desired distribution: %v", err)
	}
	return dist, r.DistRepo.Delete(dist)
}

func (r *IngressReconciler) build(distBuilder cloudfront.DistributionBuilder) (cloudfront.Distribution, error) {
	dist, err := distBuilder.Build()
	if err != nil {
		return cloudfront.Distribution{}, err
	}
	r.log.V(1).Info("Built desired distribution.", "distribution", dist)
	return dist, nil
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
	return r.Status().Update(context.Background(), status)
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

func isRemoving(obj client.Object) bool {
	return obj.GetDeletionTimestamp() != nil || (!hasGroupAnnotation(obj) && hasFinalizer(obj))
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
