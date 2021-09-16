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

type Origin struct {
	Host      string
	Behaviors []Behavior
}

type Behavior struct {
	PathPattern string
	ViewerFnARN string
}

type OriginBuilder struct {
	origin      Origin
	viewerFnARN string
}

func NewOriginBuilder(host string) OriginBuilder {
	return OriginBuilder{origin: Origin{Host: host}}
}

func (b OriginBuilder) WithBehavior(pathPattern string) OriginBuilder {
	b.origin.Behaviors = append(b.origin.Behaviors, Behavior{PathPattern: pathPattern})
	return b
}

func (b OriginBuilder) WithViewerFunction(fnARN string) OriginBuilder {
	b.viewerFnARN = fnARN
	return b
}

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
