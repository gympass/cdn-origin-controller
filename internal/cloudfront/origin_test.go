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
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRunOriginTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &OriginTestSuite{})
}

type OriginTestSuite struct {
	suite.Suite
}

func (s *OriginTestSuite) TestNewOriginBuilder_DefaultsForPublicOrigin() {
	o := NewOriginBuilder("origin", "Public").WithBehavior("/*").Build()

	s.Equal(int64(30), o.ResponseTimeout)
	s.Equal(allViewerOriginRequestPolicyID, o.Behaviors[0].RequestPolicy)
}

func (s *OriginTestSuite) TestNewOriginBuilder_DefaultsForBucketOrigin() {
	o := NewOriginBuilder("origin", "Bucket").WithBehavior("/*").Build()

	s.Equal(int64(30), o.ResponseTimeout)
	s.Equal(allViewerExceptHostHeaderOriginRequestPolicyID, o.Behaviors[0].RequestPolicy)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithBehavior_SingleBehavior() {
	o := NewOriginBuilder("origin", "Public").WithBehavior("/*").Build()
	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 1)
	s.Equal("/*", o.Behaviors[0].PathPattern)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithBehavior_MultipleBehaviors() {
	o := NewOriginBuilder("origin", "Public").
		WithBehavior("/*").
		WithBehavior("/foo").
		WithBehavior("/bar").
		Build()
	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 3)

	gotPaths := []string{
		o.Behaviors[0].PathPattern,
		o.Behaviors[1].PathPattern,
		o.Behaviors[2].PathPattern,
	}
	expectedPaths := []string{
		"/*",
		"/foo",
		"/bar",
	}

	s.ElementsMatch(expectedPaths, gotPaths)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithBehavior_DuplicatePaths() {
	o := NewOriginBuilder("origin", "Public").
		WithBehavior("/").
		WithBehavior("/").
		Build()

	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 1)
	s.Equal("/", o.Behaviors[0].PathPattern)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithViewerFunction() {
	o := NewOriginBuilder("origin", "Public").
		WithBehavior("/").
		WithBehavior("/foo").
		WithViewerFunction("some-arn").
		Build()
	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 2)
	s.Equal("some-arn", o.Behaviors[0].ViewerFnARN)
	s.Equal("some-arn", o.Behaviors[1].ViewerFnARN)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithRequestPolicy() {
	o := NewOriginBuilder("origin", "Public").
		WithBehavior("/").
		WithBehavior("/foo").
		WithRequestPolicy("some-policy").
		Build()
	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 2)
	s.Equal("some-policy", o.Behaviors[0].RequestPolicy)
	s.Equal("some-policy", o.Behaviors[1].RequestPolicy)
}
