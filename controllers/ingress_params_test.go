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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunIngressParamsTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &IngressParamsSuite{})
}

type IngressParamsSuite struct {
	suite.Suite
}

func (s *IngressParamsSuite) Test_ingressParams_V1_With_AlternativeDomainNames() {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Annotations: map[string]string{
				"cdn-origin-controller.gympass.com/cf.alternate-domain-names": "banana.com.br,banana.com.br,pera.com.br",
			},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						Hostname: "ingress.aws.load.balancer.com",
					},
				},
			},
		},
	}
	ip := newIngressParamsV1(ing)
	s.Equal([]string{"banana.com.br", "pera.com.br"}, ip.alternateDomainNames)
}

func (s *IngressParamsSuite) Test_sharedIngressParams_Valid() {
	params := []ingressParams{
		{
			group:     "foo",
			webACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/00000-5c43-4ea0-8424-2ed34dd3434",
		},
		{
			group: "foo",
		},
	}

	shared, err := newSharedIngressParams(params)

	expected := sharedIngressParams{
		webACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/00000-5c43-4ea0-8424-2ed34dd3434",
	}

	s.Equal(expected, shared)
	s.NoError(err)
}

func (s *IngressParamsSuite) Test_sharedIngressParams_ConflictingWebACLs() {
	params := []ingressParams{
		{
			group:     "foo",
			webACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/00000-5c43-4ea0-8424-2ed34dd3434",
		},
		{
			group:     "foo",
			webACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/8ab0c8f8-5c43-4ea0-8424-56dd8ab8c",
		},
	}

	shared, err := newSharedIngressParams(params)

	s.Equal(sharedIngressParams{}, shared)
	s.Error(err)
}
