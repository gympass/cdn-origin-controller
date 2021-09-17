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

// Origin represents a CloudFront Origin and aggregates Behaviors associated with it
type Origin struct {
	// Host is the origin's hostname
	Host string
	// Behaviors is the collection of Behaviors associated with this Origin
	Behaviors []Behavior
}

// Behavior represents a CloudFront Cache Behavior
type Behavior struct {
	// PathPattern is the path pattern used when configuring the Behavior
	PathPattern string
	// ViewerFnARN is the ARN of the function to be associated with the Behavior's viewer requests
	ViewerFnARN string
}

// OriginBuilder allows the construction of a Origin
type OriginBuilder struct {
	origin      Origin
	viewerFnARN string
}

// NewOriginBuilder returns an OriginBuilder for a given host
func NewOriginBuilder(host string) OriginBuilder {
	return OriginBuilder{origin: Origin{Host: host}}
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

// Build creates an Origin based on configuration made so far
func (b OriginBuilder) Build() Origin {
	if len(b.viewerFnARN) > 0 {
		b.addViewerFnToBehaviors()
	}
	return b.origin
}

func (b OriginBuilder) addViewerFnToBehaviors() {
	for i := range b.origin.Behaviors {
		b.origin.Behaviors[i].ViewerFnARN = b.viewerFnARN
	}
}
