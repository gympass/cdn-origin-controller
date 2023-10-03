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
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
)

type byMostSpecificPath []Behavior

func (s byMostSpecificPath) Len() int {
	return len(s)
}
func (s byMostSpecificPath) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byMostSpecificPath) Less(i, j int) bool {
	return isInMostSpecificOrder(s[i].PathPattern, s[j].PathPattern)
}

// isInMostSpecificOrder is true if i is longer than j
//
// If both are the same length, i is more specific if it comes before j in a
// special alphabetical order, where "*" and "?" come after all other characters.
//
// The order can be summarized as [0-9],[A-Z],[a-z], ?, *
//
// "?" is considered more specific than "*" because it represents a single char,
// while "*" represents zero or more.
func isInMostSpecificOrder(i string, j string) bool {
	if len(i) != len(j) {
		return len(i) > len(j)
	}

	// both strings are equal in length, `range i` is equivalent to `range j`
	for idx := range i {
		iChar, jChar := string(i[idx]), string(j[idx])
		switch iChar {
		case jChar: // they're equal, check next char
			continue

		case "*":
			// not out-of-order only if the other is *,
			// which we treat in the case above, must be out-of-order
			return false

		case "?":
			// only out-of-order if the other is not *, which is less specific
			return jChar == "*"

		default: // iChar not a wildcard
			if jChar == "*" || jChar == "?" {
				return true
			}
			return iChar < jChar
		}
	}

	// i and j are exactly the same, already in order
	return true
}

type byKey []*awscloudfront.Tag

func (s byKey) Len() int           { return len(s) }
func (s byKey) Less(i, j int) bool { return *s[i].Key < *s[j].Key }
func (s byKey) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
