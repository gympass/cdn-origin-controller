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

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
)

func TestRunDistributionTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DistributionTestSuite{})
}

type DistributionTestSuite struct {
	suite.Suite
}

func (s *DistributionTestSuite) TestDistribution_CustomBehaviors() {
	defaultOriginDomain := "test.default.origin"
	defaultWebACL := "default-web-acl"
	description := "test description"
	priceClass := "test price class"
	group := "test group"

	dist, err := cloudfront.NewDistributionBuilder(defaultOriginDomain, description, priceClass, group, defaultWebACL).
		WithOrigin(cloudfront.NewOriginBuilder("host").WithBehavior("/short").Build()).
		WithOrigin(cloudfront.NewOriginBuilder("host").WithBehavior("/longest").Build()).
		WithOrigin(cloudfront.NewOriginBuilder("host").WithBehavior("/longer").Build()).
		Build()
	s.NoError(err)

	expected := []cloudfront.Behavior{
		cloudfront.NewOriginBuilder("host").WithBehavior("/longest").Build().Behaviors[0],
		cloudfront.NewOriginBuilder("host").WithBehavior("/longer").Build().Behaviors[0],
		cloudfront.NewOriginBuilder("host").WithBehavior("/short").Build().Behaviors[0],
	}
	got := dist.SortedCustomBehaviors()
	s.Equal(expected, got)
}

func (s *DistributionTestSuite) TestDistributionBuilder_New() {
	defaultOriginDomain := "test.default.origin"
	defaultWebACL := "default-web-acl"
	description := "test description"
	priceClass := "test price class"
	group := "test group"

	dist, err := cloudfront.NewDistributionBuilder(defaultOriginDomain, description, priceClass, group, defaultWebACL).Build()
	s.NoError(err)
	s.Equal("test.default.origin", dist.DefaultOrigin.Host)
	s.Equal("test description", dist.Description)
	s.Equal("test price class", dist.PriceClass)
	s.Equal("default-web-acl", defaultWebACL)
	s.Equal("true", dist.Tags["cdn-origin-controller.gympass.com/owned"])
	s.Equal("test group", dist.Tags["cdn-origin-controller.gympass.com/cdn.group"])
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithOrigin() {
	defaultOriginDomain := "test.default.origin"
	defaultWebACL := "default-web-acl"
	description := "test description"
	priceClass := "test price class"
	group := "test group"
	origin := cloudfront.Origin{
		Host:            "test.custom.origin",
		Behaviors:       nil,
		ResponseTimeout: 30,
	}

	dist, err := cloudfront.NewDistributionBuilder(defaultOriginDomain, description, priceClass, group, defaultWebACL).
		WithOrigin(origin).
		Build()

	s.NoError(err)
	s.Len(dist.CustomOrigins, 1)
	s.Equal(origin, dist.CustomOrigins[0])
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithDuplicateOrigins() {
	defaultOriginDomain := "test.default.origin"
	defaultWebACL := "default-web-acl"
	description := "test description"
	priceClass := "test price class"
	group := "test group"

	dist, err := cloudfront.NewDistributionBuilder(defaultOriginDomain, description, priceClass, group, defaultWebACL).
		WithOrigin(cloudfront.NewOriginBuilder("host").WithBehavior("/path1").Build()).
		WithOrigin(cloudfront.NewOriginBuilder("host").WithBehavior("/path2").WithBehavior("/path3").Build()).
		WithOrigin(cloudfront.NewOriginBuilder("host").WithBehavior("/path4").Build()).
		Build()

	s.NoError(err)
	s.Len(dist.CustomOrigins, 1)
	s.Len(dist.CustomOrigins[0].Behaviors, 4)

	newBehavior := func(path string) cloudfront.Behavior {
		return cloudfront.Behavior{
			PathPattern:   path,
			RequestPolicy: "216adef6-5c7f-47e4-b989-5492eafa07d3",
			CachePolicy:   "4135ea2d-6df8-44a3-9df3-4b5a84be39ad",
			ViewerFnARN:   "",
			OriginHost:    "host",
		}
	}

	expectedBehaviors := []cloudfront.Behavior{
		newBehavior("/path1"),
		newBehavior("/path2"),
		newBehavior("/path3"),
		newBehavior("/path4"),
	}

	s.ElementsMatch(expectedBehaviors, dist.CustomOrigins[0].Behaviors)
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithLogging() {
	bucketAddr := "test.bucket.address"
	prefix := "test prefix"
	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithLogging(bucketAddr, prefix).
		Build()

	s.NoError(err)
	s.True(dist.Logging.Enabled)
	s.Equal("test.bucket.address", dist.Logging.BucketAddress)
	s.Equal("test prefix", dist.Logging.Prefix)
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithCustomTags() {
	tags := map[string]string{
		"testKey":  "testValue",
		"testKey2": "testValue2",
	}

	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithTags(tags).
		Build()

	s.NoError(err)
	for k, v := range tags {
		s.Equal(v, dist.Tags[k], "key: %s\tvalue: %s", k, v)
	}
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithTLS() {
	certARN := "test:arn"
	securityPolicyID := "test-policy"

	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithTLS(certARN, securityPolicyID).
		Build()

	s.NoError(err)
	s.True(dist.TLS.Enabled)
	s.Equal("test:arn", dist.TLS.CertARN)
	s.Equal("test-policy", dist.TLS.SecurityPolicyID)
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithIPv6() {
	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithIPv6().
		Build()

	s.NoError(err)
	s.True(dist.IPv6Enabled)
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithAlternateDomains() {
	domains := []string{"test.domain", "test2.domain"}

	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithAlternateDomains(domains).
		Build()

	s.NoError(err)
	s.Equal(domains, dist.AlternateDomains)
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithWebACL() {
	aclID := "test:acl"

	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithWebACL(aclID).
		Build()

	s.NoError(err)
	s.Equal("test:acl", dist.WebACLID)
}

func (s *DistributionTestSuite) TestDistributionBuilder_WithARN() {
	dist, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithARN("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA").
		Build()

	s.NoError(err)
	s.Equal("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA", dist.ARN)
	s.Equal("AAAAAAAAAAAAAA", dist.ID)
}

func (s *DistributionTestSuite) TestDistributionBuilder_InvalidDistribution() {
	origin1 := cloudfront.NewOriginBuilder("host").WithResponseTimeout(35).Build()
	origin2 := cloudfront.NewOriginBuilder("host").WithResponseTimeout(40).Build()

	_, err := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group", "default-web-acl").
		WithOrigin(origin1).
		WithOrigin(origin2).
		Build()

	s.Error(err)
}
