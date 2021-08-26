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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

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
	ing, ok := o.(*networkingv1beta1.Ingress)
	if !ok {
		return false
	}

	return len(ing.Status.LoadBalancer.Ingress) > 0
}
