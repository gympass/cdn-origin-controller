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

func (s *OriginTestSuite) TestDistributionBuilder_New() {
	defaultOriginDomain := "test.default.origin"
	description := "test description"
	priceClass := "test price class"
	group := "test group"
	origin := cloudfront.Origin{
		Host:            "test.custom.origin",
		Behaviors:       nil,
		ResponseTimeout: 30,
	}

	dist := cloudfront.NewDistributionBuilder(origin, defaultOriginDomain, description, priceClass, group).Build()
	s.Equal([]cloudfront.Origin{origin}, dist.CustomOrigins)
	s.Equal("test.default.origin", dist.DefaultOrigin.Host)
	s.Equal("test description", dist.Description)
	s.Equal("test price class", dist.PriceClass)
	s.Equal("true", dist.Tags["cdn-origin-controller.gympass.com/owned"])
	s.Equal("test group", dist.Tags["cdn-origin-controller.gympass.com/cdn.group"])
}

func (s *OriginTestSuite) TestDistributionBuilder_WithLogging() {
	bucketAddr := "test.bucket.address"
	prefix := "test prefix"
	dist := cloudfront.NewDistributionBuilder(cloudfront.Origin{}, "domain", "description", "priceClass", "group").
		WithLogging(bucketAddr, prefix).
		Build()

	s.True(dist.Logging.Enabled)
	s.Equal("test.bucket.address", dist.Logging.BucketAddress)
	s.Equal("test prefix", dist.Logging.Prefix)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithCustomTags() {
	tags := map[string]string{
		"testKey":  "testValue",
		"testKey2": "testValue2",
	}

	dist := cloudfront.NewDistributionBuilder(cloudfront.Origin{}, "domain", "description", "priceClass", "group").
		WithTags(tags).
		Build()

	for k, v := range tags {
		s.Equal(v, dist.Tags[k], "key: %s\tvalue: %s", k, v)
	}
}

func (s *OriginTestSuite) TestDistributionBuilder_WithTLS() {
	certARN := "test:arn"
	securityPolicyID := "test-policy"

	dist := cloudfront.NewDistributionBuilder(cloudfront.Origin{}, "domain", "description", "priceClass", "group").
		WithTLS(certARN, securityPolicyID).
		Build()

	s.True(dist.TLS.Enabled)
	s.Equal("test:arn", dist.TLS.CertARN)
	s.Equal("test-policy", dist.TLS.SecurityPolicyID)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithIPv6() {
	dist := cloudfront.NewDistributionBuilder(cloudfront.Origin{}, "domain", "description", "priceClass", "group").
		WithIPv6().
		Build()

	s.True(dist.IPv6Enabled)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithAlternateDomains() {
	domains := []string{"test.domain", "test2.domain"}

	dist := cloudfront.NewDistributionBuilder(cloudfront.Origin{}, "domain", "description", "priceClass", "group").
		WithAlternateDomains(domains).
		Build()

	s.Equal(domains, dist.AlternateDomains)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithWebACL() {
	aclID := "test:acl"

	dist := cloudfront.NewDistributionBuilder(cloudfront.Origin{}, "domain", "description", "priceClass", "group").
		WithWebACL(aclID).
		Build()

	s.Equal("test:acl", dist.WebACLID)
}
