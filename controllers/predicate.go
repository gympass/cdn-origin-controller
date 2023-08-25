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
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

type ingressPredicate struct{}

var _ predicate.Predicate = &ingressPredicate{}

func (p ingressPredicate) Create(event event.CreateEvent) bool {
	cdnClassOK := k8s.CDNClassNotEmpty(k8s.CDNClassAnnotationValue(event.Object))
	finalizerOK := k8s.HasFinalizer(event.Object)
	groupOK := k8s.HasGroupAnnotation(event.Object)
	lbOK := k8s.HasLoadBalancer(event.Object)

	return cdnClassOK &&
		(finalizerOK || (groupOK && lbOK))
}

func (p ingressPredicate) Delete(event event.DeleteEvent) bool {
	cdnClassOK := k8s.CDNClassNotEmpty(k8s.CDNClassAnnotationValue(event.Object))
	finalizerOK := k8s.HasFinalizer(event.Object)
	groupOK := k8s.HasGroupAnnotation(event.Object)
	lbOK := k8s.HasLoadBalancer(event.Object)

	return cdnClassOK &&
		(finalizerOK || (groupOK && lbOK))
}

func (p ingressPredicate) Update(event event.UpdateEvent) bool {
	cdnClassOK := k8s.CDNClassNotEmpty(k8s.CDNClassAnnotationValue(event.ObjectNew))
	objectsAreEqual := reflect.DeepEqual(event.ObjectNew, event.ObjectOld)
	finalizerOK := k8s.HasFinalizer(event.ObjectNew)
	groupOK := k8s.HasGroupAnnotation(event.ObjectNew)
	lbOK := k8s.HasLoadBalancer(event.ObjectNew)

	return cdnClassOK &&
		!objectsAreEqual &&
		(finalizerOK || (groupOK && lbOK))
}

func (p ingressPredicate) Generic(event.GenericEvent) bool {
	return false
}
