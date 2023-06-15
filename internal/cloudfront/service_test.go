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

func TestRunCloudFrontServiceTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CloudFrontServiceTestSuite{})
}

type CloudFrontServiceTestSuite struct {
	suite.Suite
}

func (s *CloudFrontServiceTestSuite) Test_getDeletions() {
	testCases := []struct {
		name             string
		desired, current []string
		want             []string
	}{
		{
			name:    "No current state",
			desired: []string{"foo"},
			current: nil,
			want:    nil,
		},
		{
			name:    "Current and desired state match",
			desired: []string{"foo"},
			current: []string{"foo"},
			want:    nil,
		},
		{
			name:    "Current and desired state don't match",
			desired: []string{"foo"},
			current: []string{"bar"},
			want:    []string{"bar"},
		},
		{
			name:    "No desired state and has current state",
			desired: nil,
			current: []string{"foo"},
			want:    []string{"foo"},
		},
	}

	for _, tc := range testCases {
		s.Equal(tc.want, getDeletions(tc.desired, tc.current), "test: %s", tc.name)
	}
}
