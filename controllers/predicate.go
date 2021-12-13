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

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

type ingressPredicate struct{}

var _ predicate.Predicate = &ingressPredicate{}

func (p ingressPredicate) Create(event event.CreateEvent) bool {
	cdnClassOK := cdnClassMatches(cdnClassAnnotationValue(event.Object))
	finalizerOK := hasFinalizer(event.Object)
	groupOK := hasGroupAnnotation(event.Object)
	lbOK := hasLoadBalancer(event.Object)

	return cdnClassOK &&
		(finalizerOK || (groupOK && lbOK))
}

func (p ingressPredicate) Delete(event event.DeleteEvent) bool {
	cdnClassOK := cdnClassMatches(cdnClassAnnotationValue(event.Object))
	finalizerOK := hasFinalizer(event.Object)
	groupOK := hasGroupAnnotation(event.Object)
	lbOK := hasLoadBalancer(event.Object)

	return cdnClassOK &&
		(finalizerOK || (groupOK && lbOK))
}

func (p ingressPredicate) Update(event event.UpdateEvent) bool {
	cdnClassOK := cdnClassMatches(cdnClassAnnotationValue(event.ObjectNew))
	objectsAreEqual := reflect.DeepEqual(event.ObjectNew, event.ObjectOld)
	finalizerOK := hasFinalizer(event.ObjectNew)
	groupOK := hasGroupAnnotation(event.ObjectNew)
	lbOK := hasLoadBalancer(event.ObjectNew)

	return cdnClassOK &&
		!objectsAreEqual &&
		(finalizerOK || (groupOK && lbOK))
}

func (p ingressPredicate) Generic(event.GenericEvent) bool {
	return false
}

func cdnClassAnnotationValue(object client.Object) string {
	return object.GetAnnotations()[cdnClassAnnotation]
}

func cdnClassMatches(candidate string) bool {
	return candidate == config.CDNClass()
}

func hasFinalizer(object client.Object) bool {
	return strhelper.Contains(object.GetFinalizers(), cdnFinalizer)
}

func hasGroupAnnotation(o client.Object) bool {
	return len(o.GetAnnotations()[cdnGroupAnnotation]) > 0
}

func hasLoadBalancer(o client.Object) bool {
	ingv1beta1, ok := o.(*networkingv1beta1.Ingress)
	if ok {
		return len(ingv1beta1.Status.LoadBalancer.Ingress) > 0 && len(ingv1beta1.Status.LoadBalancer.Ingress[0].Hostname) > 0
	}

	ingv1, ok := o.(*networkingv1.Ingress)
	if !ok {
		return false
	}

	return len(ingv1.Status.LoadBalancer.Ingress) > 0 && len(ingv1.Status.LoadBalancer.Ingress[0].Hostname) > 0
}
