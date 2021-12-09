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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"

	"github.com/Gympass/cdn-origin-controller/internal/config"
)

func TestRunCloudFrontTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CloudFrontSuite{})
}

type CloudFrontSuite struct {
	suite.Suite
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_EmptyIngresses() {
	dist, err := newDistributionBuilder(nil, "group", config.Config{}).Build()
	s.NoError(err)
	s.Len(dist.CustomOrigins, 0)
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_NonEmptyIngresses() {
	ingresses := []ingressParams{
		{alternateDomainNames: []string{"origin1"}},
		{alternateDomainNames: []string{"origin2", "origin3"}},
	}
	dist, err := newDistributionBuilder(ingresses, "group", config.Config{}).Build()
	s.NoError(err)
	s.Len(dist.CustomOrigins, 2)
	s.Equal([]string{"origin1", "origin2", "origin3"}, dist.AlternateDomains)
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_WithIPv6() {
	dist, err := newDistributionBuilder(nil, "group", config.Config{CloudFrontEnableIPV6: true}).Build()
	s.NoError(err)
	s.True(dist.IPv6Enabled)
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_WithTLS() {
	cfg := config.Config{
		CloudFrontCustomSSLCertARN: "arn",
		CloudFrontSecurityPolicy:   "policy",
	}
	dist, err := newDistributionBuilder(nil, "group", cfg).Build()
	s.NoError(err)
	s.True(dist.TLS.Enabled)
	s.Equal("arn", dist.TLS.CertARN)
	s.Equal("policy", dist.TLS.SecurityPolicyID)
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_WithLogging() {
	cfg := config.Config{
		CloudFrontEnableLogging: true,
		CloudFrontS3BucketLog:   "bucket",
	}
	dist, err := newDistributionBuilder(nil, "group", cfg).Build()
	s.NoError(err)
	s.True(dist.Logging.Enabled)
	s.Equal("group", dist.Logging.Prefix)
	s.Equal("bucket", dist.Logging.BucketAddress)
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_WithCustomTags() {
	cfg := config.Config{
		CloudFrontCustomTags: map[string]string{
			"foo": "bar",
			"bar": "foo",
		},
	}
	dist, err := newDistributionBuilder(nil, "group", cfg).Build()
	s.NoError(err)
	s.Equal("bar", dist.Tags["foo"])
	s.Equal("foo", dist.Tags["bar"])
}

func (s *CloudFrontSuite) Test_newDistributionBuilder_WithWAF() {
	dist, err := newDistributionBuilder(nil, "group", config.Config{CloudFrontWAFARN: "waf"}).Build()
	s.NoError(err)
	s.Equal("waf", dist.WebACLID)
}

func (s *CloudFrontSuite) Test_newOrigin_SingleBehaviorAndRule() {
	ip := ingressParams{
		loadBalancer: "origin1",
		hosts:        []string{"host1"},
		paths: []path{
			{
				pathPattern: "/",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
		},
	}

	origin := newOrigin(ip)
	s.Equal("origin1", origin.Host)
	s.Len(origin.Behaviors, 1)
	s.Equal("/", origin.Behaviors[0].PathPattern)
}

func (s *CloudFrontSuite) Test_newOrigin_MultipleBehaviorsSingleRule() {
	ip := ingressParams{
		loadBalancer: "origin1",
		hosts:        []string{"host1"},
		paths: []path{
			{
				pathPattern: "/",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
			{
				pathPattern: "/foo",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
		},
	}

	origin := newOrigin(ip)
	s.Equal("origin1", origin.Host)
	s.Len(origin.Behaviors, 2)
	s.Equal("/", origin.Behaviors[0].PathPattern)
	s.Equal("/foo", origin.Behaviors[1].PathPattern)
}
func (s *CloudFrontSuite) Test_newOrigin_MultipleBehaviorsMultipleRules() {
	ip := ingressParams{
		loadBalancer: "origin1",
		hosts:        []string{"host1"},
		paths: []path{
			{
				pathPattern: "/",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
			{
				pathPattern: "/foo",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
			{
				pathPattern: "/foo/bar",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
			{
				pathPattern: "/bar",
				pathType:    string(networkingv1beta1.PathTypeExact),
			},
		},
	}

	origin := newOrigin(ip)
	s.Equal("origin1", origin.Host)
	s.Len(origin.Behaviors, 4)
	s.Equal("/", origin.Behaviors[0].PathPattern)
	s.Equal("/foo", origin.Behaviors[1].PathPattern)
	s.Equal("/foo/bar", origin.Behaviors[2].PathPattern)
	s.Equal("/bar", origin.Behaviors[3].PathPattern)
}

// https://kubernetes.io/docs/concepts/services-networking/ingress/#examples
func (s *CloudFrontSuite) Test_newCloudFrontOrigins_PrefixPathType_SingleSlashSpecialCase() {
	ip := ingressParams{
		loadBalancer: "origin1",
		hosts:        []string{"host1"},
		paths: []path{
			{
				pathPattern: "/",
				pathType:    string(networkingv1beta1.PathTypePrefix),
			},
		},
	}

	origin := newOrigin(ip)
	s.Equal("origin1", origin.Host)
	s.Len(origin.Behaviors, 1)
	s.Equal("/*", origin.Behaviors[0].PathPattern)
}

// https://kubernetes.io/docs/concepts/services-networking/ingress/#examples
func (s *CloudFrontSuite) Test_newCloudFrontOrigins_PrefixPathType_EndsWithSlash() {
	ip := ingressParams{
		loadBalancer: "origin1",
		hosts:        []string{"host1"},
		paths: []path{
			{
				pathPattern: "/foo/",
				pathType:    string(networkingv1beta1.PathTypePrefix),
			},
		},
	}

	origin := newOrigin(ip)
	s.Equal("origin1", origin.Host)
	s.Len(origin.Behaviors, 2)
	s.Equal("/foo", origin.Behaviors[0].PathPattern)
	s.Equal("/foo/*", origin.Behaviors[1].PathPattern)
}

// https://kubernetes.io/docs/concepts/services-networking/ingress/#examples
func (s *CloudFrontSuite) Test_newCloudFrontOrigins_PrefixPathType_DoesNotEndWithSlash() {
	ip := ingressParams{
		loadBalancer: "origin1",
		hosts:        []string{"host1"},
		paths: []path{
			{
				pathPattern: "/foo",
				pathType:    string(networkingv1beta1.PathTypePrefix),
			},
		},
	}

	origin := newOrigin(ip)
	s.Equal("origin1", origin.Host)
	s.Len(origin.Behaviors, 2)
	s.Equal("/foo", origin.Behaviors[0].PathPattern)
	s.Equal("/foo/*", origin.Behaviors[1].PathPattern)
}
