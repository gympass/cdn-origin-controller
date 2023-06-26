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

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/stretchr/testify/suite"
)

func TestRunOriginTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &OriginTestSuite{})
}

type OriginTestSuite struct {
	suite.Suite
	cfg config.Config
}

func (s *OriginTestSuite) SetupTest() {
	s.cfg = config.Config{
		CloudFrontDefaultPublicOriginAccessRequestPolicyID: "216adef6-5c7f-47e4-b989-5492eafa07d3",
		CloudFrontDefaultBucketOriginAccessRequestPolicyID: "88a5eaf4-2fd4-4709-b370-b4c650ea3fcf",
	}
}

func (s *OriginTestSuite) TestNewOriginBuilder_DefaultsForPublicOrigin() {
	o := NewOriginBuilder("dist", "origin", "Public", s.cfg).WithBehavior("/*").Build()

	s.Equal("Public", o.Access)
	s.Equal(int64(30), o.ResponseTimeout)
	s.Equal(s.cfg.CloudFrontDefaultPublicOriginAccessRequestPolicyID, o.Behaviors[0].RequestPolicy)
}

func (s *OriginTestSuite) TestNewOriginBuilder_DefaultsForBucketOrigin() {
	o := NewOriginBuilder("dist", "origin", "Bucket", s.cfg).WithBehavior("/*").Build()

	s.Equal(int64(30), o.ResponseTimeout)
	s.Equal(s.cfg.CloudFrontDefaultBucketOriginAccessRequestPolicyID, o.Behaviors[0].RequestPolicy)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithBehavior_SingleBehavior() {
	o := NewOriginBuilder("dist", "origin", "Public", s.cfg).WithBehavior("/*").Build()
	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 1)
	s.Equal("/*", o.Behaviors[0].PathPattern)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithBehavior_MultipleBehaviors() {
	o := NewOriginBuilder("dist", "origin", "Public", s.cfg).
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
	o := NewOriginBuilder("dist", "origin", "Public", s.cfg).
		WithBehavior("/").
		WithBehavior("/").
		Build()

	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 1)
	s.Equal("/", o.Behaviors[0].PathPattern)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithViewerFunction() {
	o := NewOriginBuilder("dist", "origin", "Public", s.cfg).
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
	o := NewOriginBuilder("dist", "origin", "Public", s.cfg).
		WithBehavior("/").
		WithBehavior("/foo").
		WithRequestPolicy("some-policy").
		Build()
	s.Equal("origin", o.Host)
	s.Len(o.Behaviors, 2)
	s.Equal("some-policy", o.Behaviors[0].RequestPolicy)
	s.Equal("some-policy", o.Behaviors[1].RequestPolicy)
}

func (s *OriginTestSuite) TestNewOriginBuilder_WithBucketType() {
	o := NewOriginBuilder("dist", "origin", "Bucket", s.cfg).
		Build()
	s.Equal("origin", o.Host)
	s.Equal("Bucket", o.Access)
	s.Equal("dist-origin", o.OAC.Name)
	s.Equal("origin", o.OAC.OriginName)
	s.Equal("s3", o.OAC.OriginAccessControlOriginType)
}

func (s *OriginTestSuite) TestNewOriginBuilder_TestHasDifferentParameters() {
	o := NewOriginBuilder("dist", "origin", "Bucket", s.cfg).
		Build()
	o1 := NewOriginBuilder("foo", "origin", "Bucket", s.cfg).
		Build()
	o2 := NewOriginBuilder("dist", "bar", "Bucket", s.cfg).
		Build()
	o3 := NewOriginBuilder("dist", "origin", "Public", s.cfg).
		Build()

	s.False(o.HasEqualParameters(o1))
	s.False(o.HasEqualParameters(o2))
	s.False(o.HasEqualParameters(o3))
	s.True(o.HasEqualParameters(o))
}
