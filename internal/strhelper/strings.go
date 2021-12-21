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

package strhelper

// Set represents a set of strings
type Set map[string]bool

// NewSet initializes and returns a new Set
func NewSet() Set { return make(map[string]bool) }

// Add ensures a given string is part of the Set
func (ss Set) Add(s string) {
	ss[s] = true
}

// Contains returns whether s is part of the Set
func (ss Set) Contains(s string) bool {
	return ss[s]
}

// ToSlice takes all elements of the set and assigns them to a []string
func (ss Set) ToSlice() []string {
	var result []string
	for key := range ss {
		result = append(result, key)
	}
	return result
}

// Contains check if string exists in given slice
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Filter applies a given predicate function to each element of s.
// If the predicate is satisfied the element gets added to the result slice.
func Filter(s []string, predicate func(string) bool) []string {
	var filtered []string

	for _, it := range s {
		if predicate(it) {
			filtered = append(filtered, it)
		}
	}
	return filtered
}
