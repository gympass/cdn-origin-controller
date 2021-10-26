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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
	"github.com/Gympass/cdn-origin-controller/internal/config"
)

const (
	cdnGroupAnnotation          = "cdn-origin-controller.gympass.com/cdn.group"
	cfViewerFnAnnotation        = "cdn-origin-controller.gympass.com/cf.viewer-function-arn"
	cfOrigRespTimeoutAnnotation = "cdn-origin-controller.gympass.com/cf.origin-response-timeout"
)

const (
	attachOriginFailedReason  = "FailedToAttach"
	attachOriginSuccessReason = "SuccessfullyAttached"
)

var errNoAnnotation = errors.New(cdnGroupAnnotation + " annotation not present")

// IngressReconciler reconciles Ingress resources of any version
type IngressReconciler struct {
	Recorder record.EventRecorder
	Repo     cloudfront.DistributionRepository
	Config   config.Config
}

// Reconcile an Ingress resource of any version
func (r *IngressReconciler) Reconcile(obj client.Object) error {
	ip, err := newIngressParams(obj)
	if err != nil {
		return err
	}

	dist := newDistribution(newOrigin(ip), ip, r.Config)
	if len(dist.ID) > 0 {
		return r.handleUpdate(dist, obj)
	}
	return r.handleCreate(dist, obj)
}

func (r *IngressReconciler) handleUpdate(dist cloudfront.Distribution, obj client.Object) error {
	if err := r.Repo.Sync(dist); err != nil {
		return r.handleFailure(fmt.Sprintf("syncing distribution: %v", err), obj)
	}
	return r.handleSuccess(obj)
}

func (r *IngressReconciler) handleCreate(dist cloudfront.Distribution, obj client.Object) error {
	if err := r.Repo.Create(dist); err != nil {
		return r.handleFailure(fmt.Sprintf("creating distribution: %v", err), obj)
	}
	return r.handleSuccess(obj)
}

func (r *IngressReconciler) handleFailure(msg string, obj client.Object) error {
	r.Recorder.Event(obj, corev1.EventTypeWarning, attachOriginFailedReason, "Unable to reconcile CDN: "+msg)
	return errors.New(msg)
}

func (r *IngressReconciler) handleSuccess(obj client.Object) error {
	r.Recorder.Event(obj, corev1.EventTypeNormal, attachOriginSuccessReason, "Successfully reconciled CDN")
	return nil
}
