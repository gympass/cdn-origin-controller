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

package cloudfront_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
)

func TestRunCloudFrontOriginTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &OriginTestSuite{})
}

type OriginTestSuite struct {
	suite.Suite
}

func (s *OriginTestSuite) TestNewOrigins_SingleOriginAndBehavior() {
	rule := networkingv1.IngressRule{
		Host: "origin1",
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: pathTypePointer(networkingv1.PathTypeExact),
					},
				},
			},
		},
	}

	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{rule},
		},
	}

	cfOrigins := cloudfront.NewOrigins(ing)
	s.Len(cfOrigins, 1)
	s.Equal(rule.Host, cfOrigins[0].Host)
	s.Len(cfOrigins[0].Behaviors, 1)
	s.True(hostsInOrigins(ing.Spec.Rules, cfOrigins))
	s.True(pathsInBehaviors(rule.HTTP.Paths, cfOrigins[0].Behaviors))
}

func (s *OriginTestSuite) TestNewOrigins_SingleOriginMultipleBehaviors() {
	rule := networkingv1.IngressRule{
		Host: "origin1",
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: pathTypePointer(networkingv1.PathTypeExact),
					},
					{
						Path:     "/some-path",
						PathType: pathTypePointer(networkingv1.PathTypeExact),
					},
				},
			},
		},
	}

	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{rule},
		},
	}

	cfOrigins := cloudfront.NewOrigins(ing)
	s.Len(cfOrigins, 1)
	s.Len(cfOrigins[0].Behaviors, 2)
	s.True(hostsInOrigins(ing.Spec.Rules, cfOrigins))
	s.True(pathsInBehaviors(rule.HTTP.Paths, cfOrigins[0].Behaviors))
}

func (s *OriginTestSuite) TestNewCloudFrontOrigins_MultipleOrigins() {
	rule1 := networkingv1.IngressRule{
		Host: "origin1",
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: pathTypePointer(networkingv1.PathTypeExact),
					},
					{
						Path:     "/some-path",
						PathType: pathTypePointer(networkingv1.PathTypeExact),
					},
				},
			},
		},
	}

	rule2 := *rule1.DeepCopy()
	rule2.Host = "origin2"

	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{rule1, rule2},
		},
	}

	cfOrigins := cloudfront.NewOrigins(ing)
	s.Len(cfOrigins, 2)
	s.Equal(rule1.Host, cfOrigins[0].Host)
	s.Len(cfOrigins[0].Behaviors, 2)
	s.True(hostsInOrigins(ing.Spec.Rules, cfOrigins))
	for _, rule := range ing.Spec.Rules {
		s.True(pathsInBehaviors(rule.HTTP.Paths, cfOrigins[0].Behaviors))
	}
}

func (s *OriginTestSuite) TestNewCloudFrontOrigins_PrefixPathType() {
	rule := networkingv1.IngressRule{
		Host: "origin1",
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: pathTypePointer(networkingv1.PathTypePrefix),
					},
				},
			},
		},
	}

	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{rule},
		},
	}

	cfOrigins := cloudfront.NewOrigins(ing)
	s.Equal("/*", cfOrigins[0].Behaviors[0].Path)
}

func hostsInOrigins(rules []networkingv1.IngressRule, origins []cloudfront.Origin) bool {
	for _, r := range rules {
		if !hostInOrigins(r.Host, origins) {
			return false
		}
	}
	return true
}

func hostInOrigins(host string, origins []cloudfront.Origin) bool {
	var found bool
	for _, o := range origins {
		if o.Host == host {
			found = true
		}
	}
	return found
}

func pathsInBehaviors(paths []networkingv1.HTTPIngressPath, behaviors []cloudfront.Behavior) bool {
	for _, p := range paths {
		if !pathInBehaviors(p, behaviors) {
			return false
		}
	}
	return true
}

func pathInBehaviors(path networkingv1.HTTPIngressPath, behaviors []cloudfront.Behavior) bool {
	var found bool
	for _, b := range behaviors {
		if path.Path == b.Path {
			found = true
		}
	}
	return found
}

func pathTypePointer(pt networkingv1.PathType) *networkingv1.PathType {
	return &pt
}
