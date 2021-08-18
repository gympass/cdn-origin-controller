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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
)

const cdnIDAnnotation = "cdn-origin-controller.gympass.com/cdn.id"

const (
	attachOriginFailedReason  = "FailedToAttach"
	attachOriginSuccessReason = "SuccessfullyAttached"
)

// Reconciler ...
type Reconciler struct {
	client.Client

	OriginalLog logr.Logger
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	Repo        cloudfront.OriginRepository

	log logr.Logger
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

//Reconcile ...
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = r.OriginalLog.WithValues("Ingress", req.NamespacedName)

	ingress := &networkingv1.Ingress{}
	err := r.Client.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		if errors.IsNotFound(err) {
			r.log.Info("Ignoring not found Ingress.")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not fetch Ingress: %+v", err)
	}

	cfID, ok := ingress.ObjectMeta.Annotations[cdnIDAnnotation]
	if !ok {
		r.log.Info(cdnIDAnnotation + " annotation not present. Ignoring reconciliation request.")
		return ctrl.Result{}, nil
	}

	if err := r.Repo.Save(cfID, newOrigin(ingress.Spec.Rules, ingress.Status)); err != nil {
		r.Recorder.Eventf(ingress, corev1.EventTypeWarning, attachOriginFailedReason, "Unable to attach origin to CDN: saving origin: %v", err)
		return ctrl.Result{}, fmt.Errorf("saving origin: %v", err)
	}

	r.Recorder.Event(ingress, corev1.EventTypeNormal, attachOriginSuccessReason, "Successfully attached origin to CDN")
	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.And(
			&predicate.GenerationChangedPredicate{},
			&hasCdnAnnotationPredicate{},
		)).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
