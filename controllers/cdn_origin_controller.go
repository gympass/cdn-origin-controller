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
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const cdnIDAnnotation = "cdn-origin-controller.gympass.com/cdn.id"

// Reconciler ...
type Reconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

//Reconcile ...
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ingress := &networkingv1.Ingress{}

	err := r.Client.Get(ctx, req.NamespacedName, ingress)

	if err != nil {

		if errors.IsNotFound(err) {
			r.Log.Info("Ignoring not found Ingress. It can be deleted.")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not fetch Ingress: %+v", err)
	}

	fmt.Println(ingress)

	fmt.Printf("%+v", newOrigin(ingress.Spec.Rules, ingress.Status))

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

type hasCdnAnnotationPredicate struct{}

var _ predicate.Predicate = &hasCdnAnnotationPredicate{}

func (p hasCdnAnnotationPredicate) Create(event event.CreateEvent) bool {
	return hasCdnAnnotation(event.Object) && hasLoadBalancer(event.Object)
}

func (p hasCdnAnnotationPredicate) Delete(event.DeleteEvent) bool {
	return false
}

func (p hasCdnAnnotationPredicate) Update(event event.UpdateEvent) bool {
	return hasCdnAnnotation(event.ObjectNew) && hasLoadBalancer(event.ObjectNew)
}

func (p hasCdnAnnotationPredicate) Generic(event.GenericEvent) bool {
	return false
}

func hasCdnAnnotation(o client.Object) bool {
	return len(o.GetAnnotations()[cdnIDAnnotation]) > 0
}

func hasLoadBalancer(o client.Object) bool {
	ing, ok := o.(*networkingv1.Ingress)
	if !ok {
		return false
	}

	return len(ing.Status.LoadBalancer.Ingress) > 0
}
