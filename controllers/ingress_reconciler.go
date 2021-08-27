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
)

const cdnIDAnnotation = "cdn-origin-controller.gympass.com/cdn.id"

const (
	attachOriginFailedReason  = "FailedToAttach"
	attachOriginSuccessReason = "SuccessfullyAttached"
)

var errNoAnnotation = errors.New(cdnIDAnnotation + " annotation not present")

type IngressReconciler struct {
	Recorder record.EventRecorder
	Repo     cloudfront.OriginRepository
}

func (r *IngressReconciler) Reconcile(obj client.Object) error {
	cdnID, ok := obj.GetAnnotations()[cdnIDAnnotation]
	if !ok {
		return errNoAnnotation
	}

	dto, err := newIngressDTO(obj)
	if err != nil {
		return err
	}

	if err := r.Repo.Save(cdnID, newOrigin(dto)); err != nil {
		r.Recorder.Eventf(obj, corev1.EventTypeWarning, attachOriginFailedReason, "Unable to attach origin to CDN: saving origin: %v", err)
		return fmt.Errorf("saving origin: %v", err)
	}

	r.Recorder.Event(obj, corev1.EventTypeNormal, attachOriginSuccessReason, "Successfully attached origin to CDN")
	return nil
}
