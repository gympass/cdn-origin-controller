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

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRunCDNIngressTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CDNIngressSuite{})
}

type CDNIngressSuite struct {
	suite.Suite
}

func (s *CDNIngressSuite) TestNewCDNIngressFromV1_WithAlternativeDomainNames() {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Annotations: map[string]string{
				"cdn-origin-controller.gympass.com/cf.alternate-domain-names": "banana.com.br,banana.com.br,pera.com.br",
			},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{
					{
						Hostname: "ingress.aws.load.balancer.com",
					},
				},
			},
		},
	}
	ip, _ := NewCDNIngressFromV1(context.Background(), ing, CDNClass{})
	s.Equal([]string{"banana.com.br", "pera.com.br"}, ip.AlternateDomainNames)
}

func (s *CDNIngressSuite) TestNewCDNIngressFromV1_WithValidTags() {
	tags := `
area: platform
product-domain: operators
foo: bar
`
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Annotations: map[string]string{
				"cdn-origin-controller.gympass.com/cf.tags": tags,
			},
		},
	}

	expected := map[string]string{
		"area":           "platform",
		"product-domain": "operators",
		"foo":            "bar",
	}

	ip, _ := NewCDNIngressFromV1(context.Background(), ing, CDNClass{})
	s.Equal(expected, ip.Tags)
}

func (s *CDNIngressSuite) TestNewCDNIngressFromV1_UsingViewerFunctionARNOnlyIsValid() {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Annotations: map[string]string{
				cfViewerFnAnnotation: "some-arn",
			},
		},
	}

	got, err := NewCDNIngressFromV1(context.Background(), ing, CDNClass{})
	s.NoError(err)
	for _, p := range got.UnmergedPaths {
		s.Equal(ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		}, p.FunctionAssociations.ViewerRequest)
	}
}

func (s *CDNIngressSuite) TestNewCDNIngressFromV1_UsingFunctionAssociationsOnlyIsValid() {
	faYAML := `
/foo/*:
  viewerRequest:
    arn: arn:aws:cloudfront::000000000000:function/test-function-associations
    functionType: cloudfront
`

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Annotations: map[string]string{
				cfFunctionAssociationsAnnotation: faYAML,
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path: "/foo/*",
								PathType: func() *networkingv1.PathType {
									it := networkingv1.PathTypeImplementationSpecific
									return &it
								}(),
							},
						},
					}},
				},
			},
		},
	}

	got, err := NewCDNIngressFromV1(context.Background(), ing, CDNClass{})
	s.NoError(err)
	for _, p := range got.UnmergedPaths {
		s.Equal(&ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "arn:aws:cloudfront::000000000000:function/test-function-associations",
				FunctionType: FunctionTypeCloudfront,
			},
		}, p.FunctionAssociations.ViewerRequest)
	}
}

func (s *CDNIngressSuite) TestNewCDNIngressFromV1_UsingFunctionAssociationsAndViewerFunctionARNIsInvalid() {
	faYAML := `
/foo/*:
  viewerRequest:
    arn: arn:aws:cloudfront::000000000000:function/test-function-associations
    functionType: 
`

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Annotations: map[string]string{
				cfViewerFnAnnotation:             "some-arn",
				cfFunctionAssociationsAnnotation: faYAML,
			},
		},
	}

	got, err := NewCDNIngressFromV1(context.Background(), ing, CDNClass{})
	s.Error(err)
	s.Empty(got)
}

func (s *CDNIngressSuite) Test_sharedIngressParams_SingleIngressIsValid() {
	params := []CDNIngress{
		{
			NamespacedName: types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			},
			Group:             "foo",
			UnmergedWebACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/00000-5c43-4ea0-8424-2ed34dd3434",
		},
		{
			NamespacedName: types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			},
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{
							ARN: "some-arn",
						},
					},
				},
			},
		},
	}

	shared, err := NewSharedIngressParams(params)

	expected := SharedIngressParams{
		WebACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/00000-5c43-4ea0-8424-2ed34dd3434",
		paths: map[types.NamespacedName][]Path{
			types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			}: {
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn"},
					},
				},
			},
		},
	}

	s.Equal(expected, shared)
	s.NoError(err)
}

func (s *CDNIngressSuite) Test_sharedIngressParams_MultipleIngressesWithDifferentPathsIsValid() {
	params := []CDNIngress{
		{
			NamespacedName: types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			},
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{
							ARN: "some-arn",
						},
					},
				},
			},
		},
		{
			NamespacedName: types.NamespacedName{
				Namespace: "ns2",
				Name:      "ingress2",
			},
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/foo",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{
							ARN: "some-other-arn",
						},
					},
				},
			},
		},
	}

	shared, err := NewSharedIngressParams(params)

	expected := SharedIngressParams{
		paths: map[types.NamespacedName][]Path{
			types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			}: {
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn"},
					},
				},
			},
			types.NamespacedName{
				Namespace: "ns2",
				Name:      "ingress2",
			}: {
				{
					PathPattern: "/foo",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-other-arn"},
					},
				},
			},
		},
	}

	s.Equal(expected.paths, shared.paths)
	s.NoError(err)
}

func (s *CDNIngressSuite) Test_sharedIngressParams_MultipleIngressesWithSamePathButDifferentFunctionEventTypesIsValid() {
	params := []CDNIngress{
		{
			NamespacedName: types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			},
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ // NOTE: viewer response event type
							ARN: "some-arn",
						},
					},
				},
			},
		},
		{
			NamespacedName: types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			},
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						OriginResponse: &OriginFunction{ // NOTE: origin response event type
							ARN: "some-other-arn",
						},
					},
				},
			},
		},
	}

	shared, err := NewSharedIngressParams(params)

	expected := SharedIngressParams{
		paths: map[types.NamespacedName][]Path{
			types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			}: {
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn"},
						OriginResponse: &OriginFunction{
							ARN: "some-other-arn",
						},
					},
				},
			},
		},
	}

	s.Equal(expected.paths, shared.paths)
	s.NoError(err)
}

func (s *CDNIngressSuite) Test_sharedIngressParams_FunctionAssociationsForSameEventAndDifferentARNIsInvalid() {
	params := []CDNIngress{
		{
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn"},
					},
				},
			},
		},
		{
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-other-arn"},
					},
				},
			},
		},
	}

	shared, err := NewSharedIngressParams(params)

	s.Empty(shared)
	s.ErrorIs(err, errSharedParamsConflictingPaths)
}
func (s *CDNIngressSuite) Test_sharedIngressParams_FunctionAssociationsForSameEventAndDifferentFunctionTypeIsInvalid() {
	params := []CDNIngress{
		{
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{
							ARN:          "some-arn",
							FunctionType: FunctionTypeCloudfront,
						},
					},
				},
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{
							ARN:          "some-arn",
							FunctionType: FunctionTypeEdge,
						},
					},
				},
			},
		},
	}

	shared, err := NewSharedIngressParams(params)

	s.Empty(shared)
	s.ErrorIs(err, errSharedParamsConflictingPaths)
}

func (s *CDNIngressSuite) Test_sharedIngressParams_FunctionAssociationsForSameEventAndDifferentIncludeBodyIsInvalid() {
	params := []CDNIngress{
		{
			Group: "foo",
			UnmergedPaths: []Path{
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						OriginRequest: &OriginRequestFunction{
							OriginFunction: OriginFunction{
								ARN: "some-arn",
							},
							IncludeBody: true,
						},
					},
				},
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						OriginRequest: &OriginRequestFunction{
							OriginFunction: OriginFunction{
								ARN: "some-arn",
							},
							IncludeBody: false,
						},
					},
				},
			},
		},
	}

	shared, err := NewSharedIngressParams(params)

	s.Empty(shared)
	s.ErrorIs(err, errSharedParamsConflictingPaths)
}

func (s *CDNIngressSuite) Test_sharedIngressParams_ConflictingWebACLs() {
	params := []CDNIngress{
		{
			Group:             "foo",
			UnmergedWebACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/00000-5c43-4ea0-8424-2ed34dd3434",
		},
		{
			Group:             "foo",
			UnmergedWebACLARN: "arn:aws:wafv2:us-east-1:000000000000:global/webacl/foo/8ab0c8f8-5c43-4ea0-8424-56dd8ab8c",
		},
	}

	shared, err := NewSharedIngressParams(params)

	s.Equal(SharedIngressParams{}, shared)
	s.ErrorIs(err, errSharedParamsConflictingACL)
}

func (s *CDNIngressSuite) TestSharedIngressParams_PathsFromIngress() {
	shared := SharedIngressParams{
		paths: map[types.NamespacedName][]Path{
			types.NamespacedName{
				Namespace: "ns1",
				Name:      "ingress1",
			}: {
				{
					PathPattern: "/",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn1"},
						OriginResponse: &OriginFunction{
							ARN: "some-other-arn1",
						},
					},
				},
			},
			types.NamespacedName{
				Namespace: "ns2",
				Name:      "ingress2",
			}: {
				{
					PathPattern: "/foo",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn2"},
						OriginResponse: &OriginFunction{
							ARN: "some-other-arn2",
						},
					},
				},
				{
					PathPattern: "/bar",
					PathType:    "Prefix",
					FunctionAssociations: FunctionAssociations{
						ViewerResponse: &ViewerFunction{ARN: "some-arn3"},
						OriginResponse: &OriginFunction{
							ARN: "some-other-arn3",
						},
					},
				},
			},
		},
	}

	s.Empty(shared.PathsFromIngress(types.NamespacedName{Name: "I don't exist"}))

	s.ElementsMatch([]Path{
		{
			PathPattern: "/",
			PathType:    "Prefix",
			FunctionAssociations: FunctionAssociations{
				ViewerResponse: &ViewerFunction{ARN: "some-arn1"},
				OriginResponse: &OriginFunction{
					ARN: "some-other-arn1",
				},
			},
		},
	}, shared.PathsFromIngress(types.NamespacedName{Namespace: "ns1", Name: "ingress1"}))

	s.ElementsMatch([]Path{
		{
			PathPattern: "/foo",
			PathType:    "Prefix",
			FunctionAssociations: FunctionAssociations{
				ViewerResponse: &ViewerFunction{ARN: "some-arn2"},
				OriginResponse: &OriginFunction{
					ARN: "some-other-arn2",
				},
			},
		},
		{
			PathPattern: "/bar",
			PathType:    "Prefix",
			FunctionAssociations: FunctionAssociations{
				ViewerResponse: &ViewerFunction{ARN: "some-arn3"},
				OriginResponse: &OriginFunction{
					ARN: "some-other-arn3",
				},
			},
		}}, shared.PathsFromIngress(types.NamespacedName{Namespace: "ns2", Name: "ingress2"}))
}
