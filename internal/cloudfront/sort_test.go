// Copyright (c) 2023 GPBR Participacoes LTDA.
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
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRunSortTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &SortTestSuite{})
}

type SortTestSuite struct {
	suite.Suite
}

func (s *SortTestSuite) TestSort_byMostSpecificPath() {
	testCases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "Same paths",
			input: []string{"/foo", "/foo"},
			want:  []string{"/foo", "/foo"},
		},
		{
			name:  "No wildcards, different lengths",
			input: []string{"/foo", "/foo/bar"},
			want:  []string{"/foo/bar", "/foo"},
		},
		{
			name:  "No wildcards, same lengths",
			input: []string{"/bbb", "/aaa"},
			want:  []string{"/aaa", "/bbb"},
		},
		{
			name:  "With '?' wildcards, same lengths",
			input: []string{"/??-??", "/pt-br"},
			want:  []string{"/pt-br", "/??-??"},
		},
		{
			name:  "With '?' and '*' wildcards, different lengths",
			input: []string{"/??-??", "/pt-br", "/*"},
			want:  []string{"/pt-br", "/??-??", "/*"},
		},
		{
			name:  "With '?' and '*' wildcards, same lengths",
			input: []string{"/?-?", "/a-a", "/b-b", "/*-*"},
			want:  []string{"/a-a", "/b-b", "/?-?", "/*-*"},
		},
		{
			name:  "With '*' wildcards, Prefix type ingress path",
			input: []string{"/foo/*", "/foo", "/foo/bar", "/foo/bar/*"},
			want:  []string{"/foo/bar/*", "/foo/bar", "/foo/*", "/foo"},
		},
		{
			name:  "With '*' wildcards, same length as simple path",
			input: []string{"/foo/*", "/foo/a"},
			want:  []string{"/foo/a", "/foo/*"},
		},
		{
			name:  "With '*' and '?' wildcards, same length as simple path",
			input: []string{"/foo/*", "/foo/a", "/foo/?"},
			want:  []string{"/foo/a", "/foo/?", "/foo/*"},
		},
		{
			name:  "With '?' wildcards for language prefix, same length as simple path",
			input: []string{"/??-??/seguridad", "/??-??/seguridad/", "/es-es/seguridad", "/es-es/seguridad/"},
			want:  []string{"/es-es/seguridad/", "/??-??/seguridad/", "/es-es/seguridad", "/??-??/seguridad"},
		},
		{
			name:  "With '*' and '?' wildcards in the middle of the paths",
			input: []string{"/foo/*/baz", "/foo/bar/baz", "/foo/???/baz"},
			want:  []string{"/foo/bar/baz", "/foo/???/baz", "/foo/*/baz"},
		},
	}

	for _, tc := range testCases {
		var behaviors []Behavior
		for _, path := range tc.input {
			behaviors = append(behaviors, Behavior{PathPattern: path})
		}

		sort.Sort(byMostSpecificPath(behaviors))

		var got []string
		for _, b := range behaviors {
			got = append(got, b.PathPattern)
		}

		s.Equalf(tc.want, got, "test case: %s", tc.name)
	}
}
