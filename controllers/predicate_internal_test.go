package controllers

import (
	"testing"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
	baseIngress      = &networkingv1.Ingress{}
	annotatedIngress = func() *networkingv1.Ingress {
		i := baseIngress.DeepCopy()
		i.Annotations = make(map[string]string)
		i.Annotations[cdnIDAnnotation] = "some value"
		return i
	}()
	provisionedIngress = func() *networkingv1.Ingress {
		i := baseIngress.DeepCopy()
		i.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{Hostname: "some value"}}
		return i
	}()
	annotatedAndProvisionedIngress = func() *networkingv1.Ingress {
		i := baseIngress.DeepCopy()
		i.Annotations = annotatedIngress.Annotations
		i.Status = provisionedIngress.Status
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
	}

	for _, tc := range testCases {
		p := &hasCdnAnnotationPredicate{}
		s.Equal(tc.want, p.Create(tc.input))
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
	}

	for _, tc := range testCases {
		p := &hasCdnAnnotationPredicate{}
		s.Equal(tc.want, p.Update(tc.input))
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
			want:  false,
		},
	}

	for _, tc := range testCases {
		p := &hasCdnAnnotationPredicate{}
		s.Equal(tc.want, p.Delete(tc.input))
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
	}

	for _, tc := range testCases {
		p := &hasCdnAnnotationPredicate{}
		s.Equal(tc.want, p.Generic(tc.input))
	}
}
