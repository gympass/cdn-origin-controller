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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// V1beta1Reconciler reconciles v1beta1 Ingress resources
type V1beta1Reconciler struct {
	client.Client

	OriginalLog       logr.Logger
	Scheme            *runtime.Scheme
	IngressReconciler IngressReconciler

	log logr.Logger
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

//Reconcile a v1beta1 Ingress resource
func (r *V1beta1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = r.OriginalLog.WithValues("Ingress", req.NamespacedName)
	r.IngressReconciler.log = r.log

	ingress := &networkingv1beta1.Ingress{}
	err := r.Client.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			r.log.Info("Ignoring not found Ingress.")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not fetch Ingress: %+v", err)
	}

	err = r.IngressReconciler.Reconcile(ingress)
	if errors.Is(err, errNoAnnotation) {
		r.log.Error(err, "Ignoring reconciliation request")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, err
}

// SetupWithManager ...
func (r *V1beta1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(&hasCdnAnnotationPredicate{}).
		For(&networkingv1beta1.Ingress{}).
		Complete(r)
}
