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
	"testing"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestRunPredicateTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &PredicateSuite{})
}

type PredicateSuite struct {
	suite.Suite
}

var (
	baseIngress      = &networkingv1beta1.Ingress{}
	annotatedIngress = func() *networkingv1beta1.Ingress {
		i := baseIngress.DeepCopy()
		i.Annotations = make(map[string]string)
		i.Annotations[cdnGroupAnnotation] = "some value"
		return i
	}()
	provisionedIngress = func() *networkingv1beta1.Ingress {
		i := baseIngress.DeepCopy()
		i.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{Hostname: "some value"}}
		return i
	}()
	annotatedAndProvisionedIngress = func() *networkingv1beta1.Ingress {
		i := baseIngress.DeepCopy()
		i.Annotations = annotatedIngress.Annotations
		i.Status = provisionedIngress.Status
		return i
	}()
	hasFinalizerIngress = func() *networkingv1beta1.Ingress {
		i := baseIngress.DeepCopy()
		i.Finalizers = []string{cdnFinalizer}
		return i
	}()
)

func (s *PredicateSuite) Test_hasCdnAnnotationPredicate_Create() {
	testCases := []struct {
		name  string
		input event.CreateEvent
		want  bool
	}{
		{
			name:  "No annotation",
			input: event.CreateEvent{Object: baseIngress},
			want:  false,
		},
		{
			name:  "Has annotation, but not provisioned",
			input: event.CreateEvent{Object: annotatedIngress},
			want:  false,
		},
		{
			name:  "Provisioned, but no annotation",
			input: event.CreateEvent{Object: provisionedIngress},
			want:  false,
		},
		{
			name:  "Has annotation and has been provisioned",
			input: event.CreateEvent{Object: annotatedAndProvisionedIngress},
			want:  true,
		},
		{
			name:  "Has finalizer",
			input: event.CreateEvent{Object: hasFinalizerIngress},
			want:  true,
		},
	}

	for _, tc := range testCases {
		p := &ingressPredicate{}
		s.Equal(tc.want, p.Create(tc.input), "test: %s", tc.name)
	}
}

func (s *PredicateSuite) Test_hasCdnAnnotationPredicate_Update() {
	testCases := []struct {
		name  string
		input event.UpdateEvent
		want  bool
	}{
		{
			name:  "No annotation",
			input: event.UpdateEvent{ObjectNew: baseIngress},
			want:  false,
		},
		{
			name:  "Has annotation, but not provisioned",
			input: event.UpdateEvent{ObjectNew: annotatedIngress},
			want:  false,
		},
		{
			name:  "Provisioned, but no annotation",
			input: event.UpdateEvent{ObjectNew: provisionedIngress},
			want:  false,
		},
		{
			name:  "Has annotation and has been provisioned",
			input: event.UpdateEvent{ObjectNew: annotatedAndProvisionedIngress},
			want:  true,
		},
		{
			name:  "Has finalizer",
			input: event.UpdateEvent{ObjectNew: hasFinalizerIngress},
			want:  true,
		},
		{
			name:  "Old and New are equal",
			input: event.UpdateEvent{ObjectNew: hasFinalizerIngress, ObjectOld: hasFinalizerIngress},
			want:  false,
		},
	}

	for _, tc := range testCases {
		p := &ingressPredicate{}
		s.Equal(tc.want, p.Update(tc.input), "test: %s", tc.name)
	}
}

func (s *PredicateSuite) Test_hasCdnAnnotationPredicate_Delete() {
	testCases := []struct {
		name  string
		input event.DeleteEvent
		want  bool
	}{
		{
			name:  "No annotation",
			input: event.DeleteEvent{Object: baseIngress},
			want:  false,
		},
		{
			name:  "Has annotation, but not provisioned",
			input: event.DeleteEvent{Object: annotatedIngress},
			want:  false,
		},
		{
			name:  "Provisioned, but no annotation",
			input: event.DeleteEvent{Object: provisionedIngress},
			want:  false,
		},
		{
			name:  "Has annotation and has been provisioned",
			input: event.DeleteEvent{Object: annotatedAndProvisionedIngress},
			want:  true,
		},
		{
			name:  "Has finalizer",
			input: event.DeleteEvent{Object: hasFinalizerIngress},
			want:  true,
		},
	}

	for _, tc := range testCases {
		p := &ingressPredicate{}
		s.Equal(tc.want, p.Delete(tc.input), "test: %s", tc.name)
	}
}

func (s *PredicateSuite) Test_hasCdnAnnotationPredicate_Generic() {
	testCases := []struct {
		name  string
		input event.GenericEvent
		want  bool
	}{
		{
			name:  "No annotation",
			input: event.GenericEvent{Object: baseIngress},
			want:  false,
		},
		{
			name:  "Has annotation, but not provisioned",
			input: event.GenericEvent{Object: annotatedIngress},
			want:  false,
		},
		{
			name:  "Provisioned, but no annotation",
			input: event.GenericEvent{Object: provisionedIngress},
			want:  false,
		},
		{
			name:  "Has annotation and has been provisioned",
			input: event.GenericEvent{Object: annotatedAndProvisionedIngress},
			want:  false,
		},
		{
			name:  "Has finalizer",
			input: event.GenericEvent{Object: hasFinalizerIngress},
			want:  false,
		},
	}

	for _, tc := range testCases {
		p := &ingressPredicate{}
		s.Equal(tc.want, p.Generic(tc.input), "test: %s", tc.name)
	}
}

func (s *PredicateSuite) Test_hasLoadBalancer_v1Ingress() {
	var ing client.Object = &networkingv1.Ingress{}
	s.False(hasLoadBalancer(ing))
}

func (s *PredicateSuite) Test_hasLoadBalancer_notAnIngress() {
	var ing client.Object = &corev1.Service{}
	s.False(hasLoadBalancer(ing))
}
