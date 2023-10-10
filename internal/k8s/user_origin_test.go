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
	"testing"

	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestRunUserOriginTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &userOriginSuite{})
}

type userOriginSuite struct {
	suite.Suite
}

func (s *userOriginSuite) Test_cdnIngressesForUserOrigins_Success() {
	testCases := []struct {
		name            string
		annotationValue string
		expectedIngs    []CDNIngress
	}{
		{
			name:            "Has no annotation",
			annotationValue: "",
			expectedIngs:    nil,
		},
		{
			name: "Set default origin access",
			annotationValue: `
                                - host: foo.com
                                  paths:
                                    - /foo
                                    - /foo/*`,
			expectedIngs: []CDNIngress{
				{
					Group:            "group",
					LoadBalancerHost: "foo.com",
					UnmergedPaths:    []Path{{PathPattern: "/foo"}, {PathPattern: "/foo/*"}},
					OriginAccess:     "Public",
				},
			},
		},
		{
			name: "Has origin access entry",
			annotationValue: `
                                - host: foo.com
                                  paths:
                                    - /foo
                                    - /foo/*
                                  originAccess: Bucket`,
			expectedIngs: []CDNIngress{
				{
					Group:            "group",
					LoadBalancerHost: "foo.com",
					UnmergedPaths:    []Path{{PathPattern: "/foo"}, {PathPattern: "/foo/*"}},
					OriginAccess:     "Bucket",
				},
			},
		},
		{
			name: "Has a single user origin",
			annotationValue: `
                                - host: foo.com
                                  responseTimeout: 35
                                  paths:
                                    - /foo
                                    - /foo/*
                                  viewerFunctionARN: foo
                                  originRequestPolicy: None`,
			expectedIngs: []CDNIngress{
				{
					Group:             "group",
					LoadBalancerHost:  "foo.com",
					UnmergedPaths:     []Path{{PathPattern: "/foo"}, {PathPattern: "/foo/*"}},
					OriginRespTimeout: int64(35),
					OriginReqPolicy:   "None",
					OriginAccess:      "Public",
				},
			},
		},
		{
			name: "Has multiple user origins",
			annotationValue: `
                                - host: foo.com
                                  paths:
                                    - /foo
                                  viewerFunctionARN: foo
                                  originRequestPolicy: None
                                  originAccess: Bucket
                                - host: bar.com
                                  responseTimeout: 35
                                  paths:
                                    - /bar`,
			expectedIngs: []CDNIngress{
				{
					Group:            "group",
					LoadBalancerHost: "foo.com",
					UnmergedPaths:    []Path{{PathPattern: "/foo"}},
					OriginReqPolicy:  "None",
					OriginAccess:     "Bucket",
				},
				{
					Group:             "group",
					LoadBalancerHost:  "bar.com",
					UnmergedPaths:     []Path{{PathPattern: "/bar"}},
					OriginRespTimeout: int64(35),
					OriginAccess:      "Public",
				},
			},
		},
	}

	for _, tc := range testCases {
		ing := &networkingv1.Ingress{}
		if len(tc.annotationValue) > 0 {
			ing.Annotations = map[string]string{
				cfUserOriginsAnnotation: tc.annotationValue,
				CDNGroupAnnotation:      "group",
			}
		}

		got, err := cdnIngressesForUserOrigins(ing)
		s.NoError(err, "test: %s", tc.name)
		s.Equal(tc.expectedIngs, got, "test: %s", tc.name)
	}
}

func (s *userOriginSuite) Test_cdnIngressesForUserOrigins_InvalidAnnotationValue() {
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
		{
			name: "Invalid Origin Access",
			annotationValue: `
                                - host: foo.com
                                  paths:
                                    - /foo
                                    - /foo/*
                                  originAccess: invalid`,
		},
	}

	for _, tc := range testCases {
		ing := &networkingv1.Ingress{}
		ing.Annotations = map[string]string{
			cfUserOriginsAnnotation: tc.annotationValue,
			CDNGroupAnnotation:      "group",
		}

		got, err := cdnIngressesForUserOrigins(ing)
		s.Error(err, "test: %s", tc.name)
		s.Nil(got, "test: %s", tc.name)
	}
}
