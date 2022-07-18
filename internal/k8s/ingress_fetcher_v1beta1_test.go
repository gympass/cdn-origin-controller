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

func (s *IngressFetcherV1Beta1TestSuite) TestFetch_Success() {
	client := fake.NewClientBuilder().
		WithObjects(&networkingv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "name", Namespace: "namespace"},
			Status: networkingv1beta1.IngressStatus{LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						Hostname: "host",
					},
				},
			}},
		}).
		Build()

	fetcher := NewIngressFetcherV1Beta1(client)

	gotIng, err := fetcher.Fetch(context.Background(), "name", "namespace")
	s.NoError(err)
	s.NotNil(gotIng)
}

func (s *IngressFetcherV1Beta1TestSuite) TestFetch_Failure() {
	client := fake.NewClientBuilder().Build()

	fetcher := NewIngressFetcherV1Beta1(client)

	gotIng, err := fetcher.Fetch(context.Background(), "name", "namespace")
	s.Error(err)
	s.Equal(CDNIngress{}, gotIng)
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

func (s *IngressFetcherV1Beta1TestSuite) TestFetchBy_Failure() {
	client := &test.MockK8sClient{}
	client.On("List", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("mock err"))

	fetcher := NewIngressFetcherV1Beta1(client)

	predicate := func(CDNIngress) bool { return false }

	gotIngs, err := fetcher.FetchBy(context.Background(), predicate)
	s.Error(err)
	s.Nil(gotIngs)
}
