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

	"github.com/Gympass/cdn-origin-controller/internal/k8s"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	OriginAccessPublic = k8s.CFUserOriginAccessPublic
	OriginAccessBucket = k8s.CFUserOriginAccessBucket
)

const (
	defaultResponseTimeout = 30
)

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
}

// HasEqualParameters returns whether both Origins have the same parameters. It ignores differences in Behaviors
func (o Origin) HasEqualParameters(o2 Origin) bool {
	return o.Host == o2.Host && o.ResponseTimeout == o2.ResponseTimeout && o.Type == o2.Type && o.OAC == o2.OAC
}

// Behavior represents a CloudFront Cache Behavior
type Behavior struct {
	// PathPattern is the path pattern used when configuring the Behavior
	PathPattern string
	// RequestPolicy is the ID of the origin request policy to be associated with this Behavior
	RequestPolicy string
	// CachePolicy is the ID of the cache policy to be associated with this Behavior
	CachePolicy string
	// ViewerFnARN is the ARN of the function to be associated with the Behavior's viewer requests
	ViewerFnARN string
	// OriginHost the origin's host this behavior belongs to
	OriginHost string
}

// OriginBuilder allows the construction of a Origin
type OriginBuilder struct {
	host             string
	viewerFnARN      string
	requestPolicy    string
	distributionName string
	cachePolicy      string
	respTimeout      int64
	accessType       string
	paths            strhelper.Set
}

// NewOriginBuilder returns an OriginBuilder for a given host
func NewOriginBuilder(distributionName, host, accessType string) OriginBuilder {
	return OriginBuilder{
		distributionName: distributionName,
		host:             host,
		respTimeout:      defaultResponseTimeout,
		requestPolicy:    defaultRequestPolicyForType(accessType),
		cachePolicy:      cachingDisabledPolicyID,
		paths:            strhelper.NewSet(),
		accessType:       accessType,
	}
}

func defaultRequestPolicyForType(accessType string) string {
	if accessType == OriginAccessBucket {
		return allViewerExceptHostHeaderOriginRequestPolicyID
	}
	return allViewerOriginRequestPolicyID
}

// WithBehavior adds a Behavior to the Origin being built given a path pattern the Behavior should respond for
func (b OriginBuilder) WithBehavior(pathPattern string) OriginBuilder {
	b.paths.Add(pathPattern)
	return b
}

// WithViewerFunction associates a function with all viewer requests of all Behaviors in the Origin being built
func (b OriginBuilder) WithViewerFunction(fnARN string) OriginBuilder {
	b.viewerFnARN = fnARN
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

// Build creates an Origin based on configuration made so far
func (b OriginBuilder) Build() Origin {
	origin := Origin{
		Host:            b.host,
		ResponseTimeout: b.respTimeout,
	}

	origin = b.addBehaviors(origin)

	if len(b.viewerFnARN) > 0 {
		origin = b.addViewerFnToBehaviors(origin)
	}

	origin = b.addCachePolicyBehaviors(origin)
	origin = b.addRequestPolicyToBehaviors(origin)

	origin = b.addOriginAccessConfiguration(origin)

	return origin
}

func (b OriginBuilder) addBehaviors(origin Origin) Origin {
	for _, p := range b.paths.ToSlice() {
		origin.Behaviors = append(origin.Behaviors, Behavior{PathPattern: p, OriginHost: b.host})
	}
	return origin
}

func (b OriginBuilder) addViewerFnToBehaviors(origin Origin) Origin {
	for i := range origin.Behaviors {
		origin.Behaviors[i].ViewerFnARN = b.viewerFnARN
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
	if b.accessType != OriginAccessBucket {
		return origin
	}

	oacName := fmt.Sprintf("%s-%s", b.distributionName, b.host)
	origin.OAC = NewOAC(oacName, b.host)

	return origin
}
