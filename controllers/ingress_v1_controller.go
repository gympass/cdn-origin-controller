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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

// V1Reconciler reconciles v1 Ingress resources
type V1Reconciler struct {
	client.Client

	CloudFrontService *cloudfront.Service
	CDNClassFetcher   k8s.CDNClassFetcher
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/finalizers,verbs=update
// +kubebuilder:rbac:groups=cdn.gympass.com,resources=cdnstatuses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cdn.gympass.com,resources=cdnstatuses/status,verbs=get;update;patch

// Reconcile a v1 Ingress resource
func (r *V1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log, _ := logr.FromContext(ctx)
	log.Info("Starting reconciliation.")

	ingress := &networkingv1.Ingress{}
	err := r.Client.Get(ctx, req.NamespacedName, ingress)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("Ignoring not found Ingress.")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("could not fetch Ingress: %+v", err)
	}

	cdnClassName := k8s.CDNClassAnnotationValue(ingress)
	cdnClass, err := r.CDNClassFetcher.FetchByName(ctx, cdnClassName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not find CDN class (%s): %v", cdnClassName, err)
	}

	reconcilingCDNIngress, err := k8s.NewCDNIngressFromV1(ingress, cdnClass)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.CloudFrontService.Reconcile(ctx, reconcilingCDNIngress, ingress)
	if err == nil {
		log.Info("Reconciliation successful.")
	}
	return ctrl.Result{}, err
}

// SetupWithManager ...
func (r *V1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(&ingressPredicate{}).
		For(&networkingv1.Ingress{}).
		Complete(r)
}
