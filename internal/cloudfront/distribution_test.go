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

	dist := cloudfront.NewDistributionBuilder(defaultOriginDomain, description, priceClass, group).Build()
	s.Equal("test description", dist.Description)
	s.Equal("test.default.origin", dist.DefaultOriginDomain)
	s.Equal("test price class", dist.PriceClass)
	s.Equal("true", dist.Tags["cdn-origin-controller.gympass.com/owned"])
	s.Equal("test group", dist.Tags["cdn-origin-controller.gympass.com/cdn.group"])
}

func (s *OriginTestSuite) TestDistributionBuilder_WithLogging() {
	bucketAddr := "test.bucket.address"
	prefix := "test prefix"
	dist := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group").
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

	dist := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group").
		WithTags(tags).
		Build()

	for k, v := range tags {
		s.Equal(v, dist.Tags[k], "key: %s\tvalue: %s", k, v)
	}
}

func (s *OriginTestSuite) TestDistributionBuilder_WithTLS() {
	certARN := "test:arn"
	securityPolicyID := "test-policy"

	dist := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group").
		WithTLS(certARN, securityPolicyID).
		Build()

	s.True(dist.TLS.Enabled)
	s.Equal("test:arn", dist.TLS.CertARN)
	s.Equal("test-policy", dist.TLS.SecurityPolicyID)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithIPv6() {
	dist := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group").
		WithIPv6().
		Build()

	s.True(dist.IPv6Enabled)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithAlternateDomains() {
	domains := []string{"test.domain", "test2.domain"}

	dist := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group").
		WithAlternateDomains(domains).
		Build()

	s.Equal(domains, dist.AlternateDomains)
}

func (s *OriginTestSuite) TestDistributionBuilder_WithWebACL() {
	aclID := "test:acl"

	dist := cloudfront.NewDistributionBuilder("domain", "description", "priceClass", "group").
		WithWebACL(aclID).
		Build()

	s.Equal("test:acl", dist.WebACLID)
}
