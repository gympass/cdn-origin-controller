// Copyright (c) 2022 GPBR Participacoes LTDA.
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

package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Gympass/cdn-origin-controller/internal/test"
)

func TestRunIngressFetcherV1Beta1TestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &IngressFetcherV1Beta1TestSuite{})
}

type IngressFetcherV1Beta1TestSuite struct {
	suite.Suite
}

func (s *IngressFetcherV1Beta1TestSuite) TestFetchBy_Success() {
	client := fake.NewClientBuilder().
		WithLists(&networkingv1beta1.IngressList{
			Items: []networkingv1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "ingress-matching-annotation",
						Namespace:   "namespace",
						Annotations: map[string]string{CDNGroupAnnotation: "group"},
					},
					Status: networkingv1beta1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								Hostname: "host",
							},
						},
					}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-with-no-annotation",
						Namespace: "namespace",
					},
					Status: networkingv1beta1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{
								Hostname: "host",
							},
						},
					}},
				},
			},
		}).
		Build()

	fetcher := NewIngressFetcherV1Beta1(client)

	predicate := func(ing CDNIngress) bool {
		return ing.Group == "group"
	}

	gotIngs, err := fetcher.FetchBy(context.Background(), predicate)
	s.NoError(err)
	s.Len(gotIngs, 1)
	s.Equal("ingress-matching-annotation", gotIngs[0].Name)
}

func (s *IngressFetcherV1Beta1TestSuite) TestFetchBy_FailureToListIngresses() {
	client := &test.MockK8sClient{}
	client.On("List", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("mock err"))

	fetcher := NewIngressFetcherV1Beta1(client)

	predicate := func(CDNIngress) bool { return false }

	gotIngs, err := fetcher.FetchBy(context.Background(), predicate)
	s.Error(err)
	s.Nil(gotIngs)
}

func (s *IngressFetcherV1Beta1TestSuite) TestFetchBy_SuccessWithUserOrigins() {
	testCases := []struct {
		name            string
		annotationValue string
		expectedIngs    []CDNIngress
	}{
		{
			name: "Has a single user origin",
			annotationValue: `
                                - host: host
                                  responseTimeout: 35
                                  paths:
                                    - /foo
                                    - /foo/*
                                  viewerFunctionARN: foo
                                  originRequestPolicy: None`,
			expectedIngs: []CDNIngress{
				{
					NamespacedName:    types.NamespacedName{Name: "name", Namespace: "namespace"},
					Group:             "group",
					LoadBalancerHost:  "host",
					Paths:             []Path{{PathPattern: "/foo"}, {PathPattern: "/foo/*"}},
					OriginRespTimeout: int64(35),
					ViewerFnARN:       "foo",
					OriginReqPolicy:   "None",
				},
			},
		},
		{
			name: "Has multiple user origins",
			annotationValue: `
                                - host: host
                                  paths:
                                    - /foo
                                  viewerFunctionARN: foo
                                  originRequestPolicy: None
                                - host: host
                                  responseTimeout: 35
                                  paths:
                                    - /bar`,
			expectedIngs: []CDNIngress{
				{
					NamespacedName:   types.NamespacedName{Name: "name", Namespace: "namespace"},
					Group:            "group",
					LoadBalancerHost: "host",
					Paths:            []Path{{PathPattern: "/foo"}},
					OriginReqPolicy:  "None",
					ViewerFnARN:      "foo",
				},
				{
					NamespacedName:    types.NamespacedName{Name: "name", Namespace: "namespace"},
					Group:             "group",
					LoadBalancerHost:  "host",
					Paths:             []Path{{PathPattern: "/bar"}},
					OriginRespTimeout: int64(35),
				},
			},
		},
	}

	for _, tc := range testCases {
		parentIng := newIngressV1Beta1WithLB("namespace", "name", map[string]string{
			cfUserOriginsAnnotation: tc.annotationValue,
			CDNGroupAnnotation:      "group",
		})

		client := fake.NewClientBuilder().
			WithLists(&networkingv1beta1.IngressList{
				Items: []networkingv1beta1.Ingress{*parentIng},
			}).
			Build()

		fetcher := NewIngressFetcherV1Beta1(client)
		predicate := func(ing CDNIngress) bool {
			return ing.Group == "group"
		}

		expectedParentCDNIng := NewCDNIngressFromV1beta1(parentIng)
		expectedIngs := append(tc.expectedIngs, expectedParentCDNIng)

		gotIngs, err := fetcher.FetchBy(context.Background(), predicate)
		s.NoError(err, "test: %s", tc.name)
		s.ElementsMatch(expectedIngs, gotIngs, "test: %s", tc.name)
	}
}

func (s *IngressFetcherV1Beta1TestSuite) TestFetchBy_FailureWithUserOrigins() {
	testCases := []struct {
		name            string
		annotationValue string
	}{
		{
			name:            "No path",
			annotationValue: "- host: foo.com",
		},
		{
			name:            "No host",
			annotationValue: "- paths: [/bar]",
		},
		{
			name:            "Invalid YAML",
			annotationValue: "*",
		},
	}

	for _, tc := range testCases {
		ing := newIngressV1Beta1WithLB("namespace", "name", map[string]string{
			cfUserOriginsAnnotation: tc.annotationValue,
			CDNGroupAnnotation:      "group",
		})

		client := fake.NewClientBuilder().
			WithLists(&networkingv1beta1.IngressList{
				Items: []networkingv1beta1.Ingress{*ing},
			}).
			Build()

		fetcher := NewIngressFetcherV1Beta1(client)
		predicate := func(ing CDNIngress) bool { return true }

		got, err := fetcher.FetchBy(context.Background(), predicate)
		s.Error(err, "test: %s", tc.name)
		s.Nil(got, "test: %s", tc.name)
	}
}

func newIngressV1Beta1WithLB(namespace, name string, annotations map[string]string) *networkingv1beta1.Ingress {
	return &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Status: networkingv1beta1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{
					Hostname: "host",
				},
			},
		}},
	}
}
