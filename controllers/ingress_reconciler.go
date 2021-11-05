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

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/config"
)

const (
	cdnGroupAnnotation               = "cdn-origin-controller.gympass.com/cdn.group"
	cfViewerFnAnnotation             = "cdn-origin-controller.gympass.com/cf.viewer-function-arn"
	cfOrigRespTimeoutAnnotation      = "cdn-origin-controller.gympass.com/cf.origin-response-timeout"
	cfAlternateDomainNamesAnnotation = "cdn-origin-controller.gympass.com/cf.alternate-domain-names"
)

const (
	attachOriginFailedReason  = "FailedToAttach"
	attachOriginSuccessReason = "SuccessfullyAttached"
)

var errNoAnnotation = errors.New(cdnGroupAnnotation + " annotation not present")

type boundIngressesFunc func(v1alpha1.CDNStatus) ([]ingressParams, error)

// IngressReconciler reconciles Ingress resources of any version
type IngressReconciler struct {
	client.Client

	BoundIngressParamsFn boundIngressesFunc
	Config               config.Config
	Recorder             record.EventRecorder
	Repo                 cloudfront.DistributionRepository

	log logr.Logger
}

// Reconcile an Ingress resource of any version
func (r *IngressReconciler) Reconcile(reconciling ingressParams, obj client.Object) error {
	cdnStatus := &v1alpha1.CDNStatus{}
	nsName := types.NamespacedName{Name: reconciling.group}
	dist := newDistribution(newOrigin(reconciling), reconciling, r.Config)

	err := r.Get(context.Background(), nsName, cdnStatus)
	if k8serrors.IsNotFound(err) {
		dist, err := r.Repo.Create(dist)
		if err != nil {
			return r.handleFailure(err, obj)
		}
		if err := r.createCDNStatus(dist, obj, reconciling.group); err != nil {
			return r.handleFailure(err, obj)
		}
		return r.handleSuccess(obj)
	}

	if err != nil {
		return fmt.Errorf("fetching CDN status: %v", err)
	}

	boundIngressesParams, err := r.BoundIngressParamsFn(filterIngressRef(cdnStatus, obj))
	if err != nil {
		return err
	}
	for _, ip := range boundIngressesParams {
		dist.AddOrigin(newOrigin(ip))
		dist.AddAlternateDomains(ip.alternateDomainNames)
	}

	dist.ID = cdnStatus.Status.ID
	dist.ARN = cdnStatus.Status.ARN
	inSync := true
	if err := r.Repo.Sync(dist); err != nil {
		inSync = false
		_ = r.updateCDNStatus(cdnStatus, inSync, dist, obj)
		return r.handleFailure(err, obj)
	}

	if err := r.updateCDNStatus(cdnStatus, inSync, dist, obj); err != nil {
		return r.handleFailure(err, obj)
	}
	return r.handleSuccess(obj)
}

func (r *IngressReconciler) createCDNStatus(dist cloudfront.Distribution, obj client.Object, group string) error {
	cdnStatus := v1alpha1.CDNStatus{
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

	if err := r.Create(context.Background(), &cdnStatus); err != nil {
		return fmt.Errorf("creating CDNStatus resource: %v", err)
	}

	const inSync = true
	return r.updateCDNStatus(&cdnStatus, inSync, dist, obj)
}

func (r *IngressReconciler) updateCDNStatus(status *v1alpha1.CDNStatus, sync bool, dist cloudfront.Distribution, obj client.Object) error {
	status.SetIngressRef(sync, obj)
	status.Status.Aliases = dist.AlternateDomains
	if err := r.Status().Update(context.Background(), status); err != nil {
		r.log.Error(err, "Could not persist CDNStatus resource", "CDNStatus", status)
		return err
	}
	return nil
}

func (r *IngressReconciler) handleFailure(err error, obj client.Object) error {
	r.Recorder.Event(obj, corev1.EventTypeWarning, attachOriginFailedReason, "Unable to reconcile CDN: "+err.Error())
	return err
}

func (r *IngressReconciler) handleSuccess(obj client.Object) error {
	r.Recorder.Event(obj, corev1.EventTypeNormal, attachOriginSuccessReason, "Successfully reconciled CDN")
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
