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

package cloudfront

// Distribution represents a CloudFront distribution
type Distribution struct {
	ID               string
	ARN              string
	Address          string
	AlternateDomains []string
	CustomOrigin     Origin
	DefaultOrigin    Origin
	Description      string
	IPv6Enabled      bool
	Logging          loggingConfig
	PriceClass       string
	Tags             map[string]string
	TLS              tlsConfig
	WebACLID         string
}

type tlsConfig struct {
	Enabled          bool
	CertARN          string
	SecurityPolicyID string
}

type loggingConfig struct {
	Enabled       bool
	BucketAddress string
	Prefix        string
}

// DistributionBuilder allows the construction of a Distribution
type DistributionBuilder struct {
	alternateDomains    []string
	customOrigin        Origin
	defaultOriginDomain string
	description         string
	ipv6Enabled         bool
	group               string
	logging             loggingConfig
	priceClass          string
	tags                map[string]string
	tls                 tlsConfig
	webACLID            string
}

// NewDistributionBuilder takes required arguments for a distribution and returns a DistributionBuilder
func NewDistributionBuilder(origin Origin, defaultOriginDomain, description, priceClass, group string) DistributionBuilder {
	return DistributionBuilder{
		customOrigin:        origin,
		description:         description,
		defaultOriginDomain: defaultOriginDomain,
		priceClass:          priceClass,
		group:               group,
	}
}

// WithLogging takes in bucket address and file prefix to enable sending CF logs to S3
func (b DistributionBuilder) WithLogging(bucketAddress, prefix string) DistributionBuilder {
	b.logging = loggingConfig{
		Enabled:       true,
		BucketAddress: bucketAddress,
		Prefix:        prefix,
	}
	return b
}

// WithTags takes in custom tags which should be present at the Distribution
func (b DistributionBuilder) WithTags(tags map[string]string) DistributionBuilder {
	b.tags = tags
	return b
}

// WithTLS takes in an ACM certificate ARN and a Security Policy ID to enable TLS termination
func (b DistributionBuilder) WithTLS(certARN, securityPolicyID string) DistributionBuilder {
	b.tls = tlsConfig{
		Enabled:          true,
		CertARN:          certARN,
		SecurityPolicyID: securityPolicyID,
	}
	return b
}

// WithIPv6 enables IPv6
func (b DistributionBuilder) WithIPv6() DistributionBuilder {
	b.ipv6Enabled = true
	return b
}

// WithAlternateDomains takes a slice of domains to be configured as the Distribution's alternate domains
func (b DistributionBuilder) WithAlternateDomains(domains []string) DistributionBuilder {
	b.alternateDomains = domains
	return b
}

// WithWebACL takes the ID of the Web ACL that should be associated with the Distribution
func (b DistributionBuilder) WithWebACL(id string) DistributionBuilder {
	b.webACLID = id
	return b
}

// Build constructs a Distribution taking into account all configuration set by previous "With*" method calls
func (b DistributionBuilder) Build() Distribution {
	return Distribution{
		CustomOrigin:     b.customOrigin,
		DefaultOrigin:    NewOriginBuilder(b.defaultOriginDomain).Build(),
		Description:      b.description,
		PriceClass:       b.priceClass,
		Tags:             b.generateTags(),
		Logging:          b.logging,
		TLS:              b.tls,
		IPv6Enabled:      b.ipv6Enabled,
		AlternateDomains: b.alternateDomains,
		WebACLID:         b.webACLID,
	}
}

func (b DistributionBuilder) generateTags() map[string]string {
	tags := b.defaultTags()
	for k, v := range b.tags {
		tags[k] = v
	}
	return tags
}

const (
	managedByTagKey   = "cdn-origin-controller.gympass.com/owned"
	managedByTagValue = "true"
	groupTagKey       = "cdn-origin-controller.gympass.com/cdn.group"
)

func (b DistributionBuilder) defaultTags() map[string]string {
	tags := make(map[string]string)
	tags[managedByTagKey] = managedByTagValue
	tags[groupTagKey] = b.group
	return tags
}
