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
}

func (o Origin) HasEqualParameters(o2 Origin) bool {
	return o.Host == o2.Host && o.ResponseTimeout == o2.ResponseTimeout
}

// Behavior represents a CloudFront Cache Behavior
type Behavior struct {
	// PathPattern is the path pattern used when configuring the Behavior
	PathPattern string
	// RequestPolicy is the ID of the origin request policy to be associated with this Behavior
	RequestPolicy string
	// ViewerFnARN is the ARN of the function to be associated with the Behavior's viewer requests
	ViewerFnARN string
}

// OriginBuilder allows the construction of a Origin
type OriginBuilder struct {
	origin        Origin
	viewerFnARN   string
	requestPolicy string
	respTimeout   int64
}

// NewOriginBuilder returns an OriginBuilder for a given host
func NewOriginBuilder(host string) OriginBuilder {
	return OriginBuilder{origin: Origin{Host: host, ResponseTimeout: defaultResponseTimeout}, requestPolicy: allViewerOriginRequestPolicyID}
}

// WithBehavior adds a Behavior to the Origin being built given a path pattern the Behavior should respond for
func (b OriginBuilder) WithBehavior(pathPattern string) OriginBuilder {
	b.origin.Behaviors = append(b.origin.Behaviors, Behavior{PathPattern: pathPattern})
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

// WithResponseTimeout associates a custom response timeout to custom origin
func (b OriginBuilder) WithResponseTimeout(rpTimeout int64) OriginBuilder {
	b.respTimeout = rpTimeout
	return b
}

// Build creates an Origin based on configuration made so far
func (b OriginBuilder) Build() Origin {
	if len(b.viewerFnARN) > 0 {
		b.addViewerFnToBehaviors()
	}

	b.addRequestPolicyToBehaviors()

	if b.respTimeout > 0 {
		b.origin.ResponseTimeout = b.respTimeout
	}
	return b.origin
}

func (b OriginBuilder) addViewerFnToBehaviors() {
	for i := range b.origin.Behaviors {
		b.origin.Behaviors[i].ViewerFnARN = b.viewerFnARN
	}
}

func (b OriginBuilder) addRequestPolicyToBehaviors() {
	for i := range b.origin.Behaviors {
		b.origin.Behaviors[i].RequestPolicy = b.requestPolicy
	}
}
