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
	"strings"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

const (
	OriginAccessPublic = k8s.CFUserOriginAccessPublic
	OriginAccessBucket = k8s.CFUserOriginAccessBucket
)

const (
	defaultResponseTimeout    = 30
	templateOriginHeadersHost = "{{origin.host}}"
)

// originHeaders represents pairs of HTTP header key/values that should be added to requests to the origin
type originHeaders struct {
	originHost string
	headers    map[string]string
}

func newOriginHeaders(originHost string, headers map[string]string) originHeaders {
	return originHeaders{
		originHost: originHost,
		headers:    headers,
	}
}

func (h originHeaders) get() map[string]string {
	if h.headers == nil {
		return nil
	}

	result := make(map[string]string, len(h.headers))
	for k, v := range h.headers {
		v = strings.ReplaceAll(v, templateOriginHeadersHost, h.originHost)
		result[k] = v
	}

	return result
}

// Origin represents a CloudFront Origin and aggregates Behaviors associated with it
type Origin struct {
	// Host is the origin's hostname
	Host string
	// Behaviors is the collection of Behaviors associated with this Origin
	Behaviors []Behavior
	// ResponseTimeout is how long CloudFront will wait for a response from the Origin in seconds
	ResponseTimeout int64
	// Access is this Origin's access type (Bucket or Public)
	Access string
	// OAC configures Access Origin Control for this Origin
	OAC OAC

	headers originHeaders
}

// HasEqualParameters returns whether both Origins have the same parameters. It ignores differences in Behaviors
func (o Origin) HasEqualParameters(o2 Origin) bool {
	return o.Host == o2.Host && o.ResponseTimeout == o2.ResponseTimeout && o.Access == o2.Access && o.OAC == o2.OAC
}

// Headers returns the headers that are bound to this Origin.
// The stored value may be changed in order to use values that are known
// only at runtime, such as the Origin's host.
func (o Origin) Headers() map[string]string {
	return o.headers.get()
}

func (o Origin) isBucketBased() bool {
	return o.Access == OriginAccessBucket
}

// Behavior represents a CloudFront Cache Behavior
type Behavior struct {
	// PathPattern is the path pattern used when configuring the Behavior
	PathPattern string
	// RequestPolicy is the ID of the origin request policy to be associated with this Behavior
	RequestPolicy string
	// CachePolicy is the ID of the cache policy to be associated with this Behavior
	CachePolicy string
	// OriginHost the origin's host this behavior belongs to
	OriginHost string
	// FunctionAssociations is a slice of Function that should be bound to this Behavior
	FunctionAssociations []Function
}

// OriginBuilder allows the construction of an Origin
type OriginBuilder struct {
	host             string
	headers          map[string]string
	requestPolicy    string
	distributionName string
	cachePolicy      string
	respTimeout      int64
	accessType       string
	behaviors        map[string][]Function
}

// NewOriginBuilder returns an OriginBuilder for a given host
func NewOriginBuilder(distributionName, host, accessType string, cfg config.Config) OriginBuilder {
	return OriginBuilder{
		distributionName: distributionName,
		host:             host,
		respTimeout:      defaultResponseTimeout,
		requestPolicy:    defaultRequestPolicyForType(accessType, cfg),
		cachePolicy:      cfg.CloudFrontDefaultCachingPolicyID,
		behaviors:        make(map[string][]Function),
		accessType:       accessType,
	}
}

func defaultRequestPolicyForType(accessType string, cfg config.Config) string {
	if accessType == OriginAccessBucket {
		return cfg.CloudFrontDefaultBucketOriginAccessRequestPolicyID
	}
	return cfg.CloudFrontDefaultPublicOriginAccessRequestPolicyID
}

// WithBehavior adds a Behavior to the Origin being built given a path pattern the Behavior should respond for.
// Also receives optional functions that should be associated to this behavior.
func (b OriginBuilder) WithBehavior(pathPattern string, functions ...Function) OriginBuilder {
	_, behaviorExists := b.behaviors[pathPattern]
	if !behaviorExists {
		b.behaviors[pathPattern] = functions
		return b
	}

	b.behaviors[pathPattern] = append(b.behaviors[pathPattern], functions...)
	return b
}

// WithRequestPolicy associates a given origin request policy ID with all Behaviors in the Origin being built
func (b OriginBuilder) WithRequestPolicy(policy string) OriginBuilder {
	if len(policy) > 0 {
		b.requestPolicy = policy
	}
	return b
}

// WithCachePolicy associates a given cache policy ID with all Behaviors in the Origin being built
func (b OriginBuilder) WithCachePolicy(policy string) OriginBuilder {
	if len(policy) > 0 {
		b.cachePolicy = policy
	}
	return b
}

// WithResponseTimeout associates a custom response timeout to custom origin
func (b OriginBuilder) WithResponseTimeout(rpTimeout int64) OriginBuilder {
	if rpTimeout > 0 {
		b.respTimeout = rpTimeout
	}
	return b
}

// WithOriginHeaders associates a map of HTTP headers that CloudFront should add on every request to the Origin
func (b OriginBuilder) WithOriginHeaders(headers map[string]string) OriginBuilder {
	b.headers = headers
	return b
}

// Build creates an Origin based on configuration made so far
func (b OriginBuilder) Build() Origin {
	origin := Origin{
		Host:            b.host,
		ResponseTimeout: b.respTimeout,
	}

	origin.headers = newOriginHeaders(b.host, b.headers)

	origin = b.addBehaviors(origin)

	origin = b.addCachePolicyBehaviors(origin)
	origin = b.addRequestPolicyToBehaviors(origin)

	origin.Access = b.accessType

	origin = b.addOriginAccessConfiguration(origin)

	return origin
}

func (b OriginBuilder) addBehaviors(origin Origin) Origin {
	for p, functions := range b.behaviors {
		origin.Behaviors = append(origin.Behaviors, Behavior{PathPattern: p, OriginHost: b.host, FunctionAssociations: functions})
	}
	return origin
}

func (b OriginBuilder) addRequestPolicyToBehaviors(origin Origin) Origin {
	for i := range origin.Behaviors {
		origin.Behaviors[i].RequestPolicy = b.requestPolicy
	}
	return origin
}

func (b OriginBuilder) addCachePolicyBehaviors(origin Origin) Origin {
	for i := range origin.Behaviors {
		origin.Behaviors[i].CachePolicy = b.cachePolicy
	}
	return origin
}

func (b OriginBuilder) addOriginAccessConfiguration(origin Origin) Origin {
	if origin.Access != OriginAccessBucket {
		return origin
	}

	origin.OAC = NewOAC(b.distributionName, b.host)
	return origin
}
