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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

// V1Reconciler reconciles v1 Ingress resources
type V1Reconciler struct {
	client.Client

	OriginalLog       logr.Logger
	Scheme            *runtime.Scheme
	IngressReconciler *IngressReconciler

	log logr.Logger
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update
// +kubebuilder:rbac:groups=cdn.gympass.com,resources=cdnstatuses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cdn.gympass.com,resources=cdnstatuses/status,verbs=get;update;patch

//Reconcile a v1 Ingress resource
func (r *V1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = r.OriginalLog.WithValues("Ingress", req.NamespacedName)
	r.IngressReconciler.log = r.log
	r.log.Info("Starting reconciliation.")

	ingress := &networkingv1.Ingress{}
	err := r.Client.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			r.log.Info("Ignoring not found Ingress.")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not fetch Ingress: %+v", err)
	}

	reconcilingIP := k8s.NewCDNIngressFromV1(ingress)
	err = r.IngressReconciler.Reconcile(reconcilingIP, ingress)
	if err == nil {
		r.log.Info("Reconciliation successful.")
	}
	return ctrl.Result{}, err
}

// BoundIngresses returns a slice of k8s.CDNIngress for each Ingress associated with a particular v1alpha1.CDNStatus
//revive:disable-next-line:unexported-return
func (r *V1Reconciler) BoundIngresses(status v1alpha1.CDNStatus) ([]k8s.CDNIngress, error) {
	var paramsList []k8s.CDNIngress
	for _, key := range status.GetIngressKeys() {
		ing := &networkingv1.Ingress{}
		err := r.Client.Get(context.Background(), key, ing)
		if err != nil {
			// @TODO: handle not found, implying the Ingress has been deleted
			// and should no longer be part of the status or distribution.
			return nil, fmt.Errorf("fetching ingress %s: %v", key.String(), err)
		}
		r.log.V(1).Info("Fetched bound Ingress", "name", ing.Name, "namespace", ing.Namespace)

		reconcilingCDNIngress := k8s.NewCDNIngressFromV1(ing)
		paramsList = append(paramsList, reconcilingCDNIngress)

		userOriginParamsList, err := r.IngressReconciler.cdnIngressesForUserOrigins(reconcilingCDNIngress.Group, ing)
		if err != nil {
			return nil, fmt.Errorf("creating user origins desired state: %v", err)
		}
		paramsList = append(paramsList, userOriginParamsList...)
	}
	return paramsList, nil
}

// SetupWithManager ...
func (r *V1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(&ingressPredicate{}).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
