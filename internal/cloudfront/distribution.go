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

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

// Distribution represents a CloudFront distribution
type Distribution struct {
	ID               string
	ARN              string
	Address          string
	AlternateDomains []string
	CustomOrigins    []Origin
	DefaultOrigin    Origin
	Description      string
	Group            string
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

// SortedCustomBehaviors returns a slice of all custom Behavior sorted by descending path length
func (d Distribution) SortedCustomBehaviors() []Behavior {
	var result []Behavior
	for _, o := range d.CustomOrigins {
		result = append(result, o.Behaviors...)
	}

	sort.Sort(byDescendingPathLength(result))
	return result
}

// Exists returns whether this Distribution exists on AWS or not
func (d Distribution) Exists() bool {
	return len(d.ID) > 0
}

// IsEmpty return whether this Distribution has custom origins/behaviors
func (d Distribution) IsEmpty() bool {
	return len(d.CustomOrigins) == 0
}

// DistributionBuilder allows the construction of a Distribution
type DistributionBuilder struct {
	id                  string
	address             string
	alternateDomains    []string
	arn                 string
	customOrigins       []Origin
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
func NewDistributionBuilder(defaultOriginDomain, description, priceClass, group, defaultWebACLID string) DistributionBuilder {
	return DistributionBuilder{
		description:         description,
		defaultOriginDomain: defaultOriginDomain,
		priceClass:          priceClass,
		group:               group,
		webACLID:            defaultWebACLID,
	}
}

// WithOrigin takes in an Origin that should be part of the Distribution
func (b DistributionBuilder) WithOrigin(o Origin) DistributionBuilder {
	b.customOrigins = append(b.customOrigins, o)
	return b
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

// AppendTags takes in custom tags which should be present at the Distribution
func (b DistributionBuilder) AppendTags(tags map[string]string) DistributionBuilder {
	b.tags = strhelper.MergeMapString(b.tags, tags)
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

// WithAlternateDomains takes a slice of domains to be added to the Distribution's alternate domains
func (b DistributionBuilder) WithAlternateDomains(domains []string) DistributionBuilder {
	for _, domain := range domains {
		if !strhelper.Contains(b.alternateDomains, domain) {
			b.alternateDomains = append(b.alternateDomains, domain)
		}
	}
	return b
}

// WithWebACL takes the ID of the Web ACL that should be associated with the Distribution
func (b DistributionBuilder) WithWebACL(id string) DistributionBuilder {
	b.webACLID = id
	return b
}

// WithARN takes in identifying information from an existing CloudFront to populate the resulting Distribution
func (b DistributionBuilder) WithARN(arn string) DistributionBuilder {
	b.id = b.extractID(arn)
	b.arn = arn
	return b
}

// extractID assumes a valid ARN is given
// arn:aws:cloudfront::<account>:distribution/<ID>
func (b DistributionBuilder) extractID(arn string) string {
	return strings.Split(arn, "/")[1]
}

// Build constructs a Distribution taking into account all configuration set by previous "With*" method calls
func (b DistributionBuilder) Build() (Distribution, error) {
	d := Distribution{
		ID:               b.id,
		ARN:              b.arn,
		Address:          b.address,
		CustomOrigins:    b.customOrigins,
		DefaultOrigin:    NewOriginBuilder(b.defaultOriginDomain).Build(),
		Description:      b.description,
		Group:            b.group,
		PriceClass:       b.priceClass,
		Tags:             b.generateTags(),
		Logging:          b.logging,
		TLS:              b.tls,
		IPv6Enabled:      b.ipv6Enabled,
		AlternateDomains: b.alternateDomains,
		WebACLID:         b.webACLID,
	}

	if err := validate(d); err != nil {
		return Distribution{}, err
	}
	return mergeCustomOrigins(d), nil
}

func (b DistributionBuilder) generateTags() map[string]string {
	tags := b.defaultTags()
	for k, v := range b.tags {
		tags[k] = v
	}
	return tags
}

const (
	ownershipTagKey   = "cdn-origin-controller.gympass.com/owned"
	ownershipTagValue = "true"
	groupTagKey       = "cdn-origin-controller.gympass.com/cdn.group"
)

func (b DistributionBuilder) defaultTags() map[string]string {
	tags := make(map[string]string)
	tags[ownershipTagKey] = ownershipTagValue
	tags[groupTagKey] = b.group
	return tags
}

func validate(d Distribution) error {
	var origins []Origin
	origins = append(origins, d.DefaultOrigin)
	origins = append(origins, d.CustomOrigins...)

	existingOrigins := make(map[string]Origin)
	for _, o := range origins {
		if existing, ok := existingOrigins[o.Host]; ok {
			if !existing.HasEqualParameters(o) {
				return fmt.Errorf("same host (%s) specified twice with different parameters for origin configuration", o.Host)
			}
		}
		existingOrigins[o.Host] = o
	}
	return nil
}

// mergeCustomOrigins ensures duplicate origins are merged into a single one.
// It expects a valid Distribution as input.
func mergeCustomOrigins(d Distribution) Distribution {
	merged := mergedOriginMap(d)

	var normalized []Origin
	for _, origin := range merged {
		normalized = append(normalized, origin)
	}

	d.CustomOrigins = normalized
	return d
}

func mergedOriginMap(d Distribution) map[string]Origin {
	existingOrigins := make(map[string]Origin) // map[host]origin
	for _, candidateO := range d.CustomOrigins {
		existingO, ok := existingOrigins[candidateO.Host]
		if !ok {
			existingOrigins[candidateO.Host] = candidateO
		} else {
			existingO.Behaviors = append(existingO.Behaviors, candidateO.Behaviors...)
			existingOrigins[existingO.Host] = existingO
		}
	}
	return existingOrigins
}
