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

package controllers

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRunUserOriginTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &userOriginSuite{})
}

type userOriginSuite struct {
	suite.Suite
}

func (s *userOriginSuite) Test_userOriginsFromYAML_Success() {
	testCases := []struct {
		name     string
		data     string
		expected []userOrigin
	}{
		{
			name: "Has a single user origin",
			data: `
                  - host: foo.com
                    responseTimeout: 35
                    paths:
                      - /foo
                      - /foo/*
                    viewerFunctionARN: foo
                    originRequestPolicy: None`,
			expected: []userOrigin{
				{
					Host:              "foo.com",
					ResponseTimeout:   35,
					Paths:             []string{"/foo", "/foo/*"},
					ViewerFunctionARN: "foo",
					RequestPolicy:     "None",
				},
			},
		},
		{
			name: "Has multiple user origins",
			data: `
                - host: foo.com
                  paths:
                    - /foo
                  viewerFunctionARN: foo
                  originRequestPolicy: None
                - host: bar.com
                  responseTimeout: 35
                  paths:
                    - /bar`,
			expected: []userOrigin{
				{
					Host:              "foo.com",
					Paths:             []string{"/foo"},
					ViewerFunctionARN: "foo",
					RequestPolicy:     "None",
				},
				{
					Host:            "bar.com",
					ResponseTimeout: 35,
					Paths:           []string{"/bar"},
				},
			},
		},
	}

	for _, tc := range testCases {
		got, err := userOriginsFromYAML([]byte(tc.data))
		s.NoError(err, "test: %s", tc.name)
		s.Equal(tc.expected, got, "test: %s", tc.name)
	}
}

func (s *userOriginSuite) Test_userOriginsFromYAML_InvalidYAMLData() {
	_, err := userOriginsFromYAML([]byte("*"))
	s.Error(err)
}

func (s *userOriginSuite) Test_userOriginsFromYAML_InvalidUserOrigin() {
	testCases := []struct {
		name string
		data string
	}{
		{
			name: "No host",
			data: `
                  - responseTimeout: 35
                    paths:
                      - /foo
                      - /foo/*
                    viewerFunctionARN: foo
                    originRequestPolicy: None`,
		},
		{
			name: "No paths",
			data: `
                - host: foo.com
                  viewerFunctionARN: foo
                  originRequestPolicy: None`,
		},
	}

	for _, tc := range testCases {
		_, err := userOriginsFromYAML([]byte(tc.data))
		s.Error(err, "test: %s", tc.name)
	}
}
