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
	"strings"

	networkingv1beta1 "k8s.io/api/networking/v1beta1"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
)

const prefixPathType = string(networkingv1beta1.PathTypePrefix)

func newOrigin(dto ingressDTO) cloudfront.Origin {
	builder := cloudfront.NewOriginBuilder(dto.host).WithViewerFunction(dto.viewerFnARN)
	patterns := pathPatterns(dto.paths)
	for _, p := range patterns {
		builder = builder.WithBehavior(p)
	}

	return builder.Build()
}

func pathPatterns(paths []path) []string {
	var patterns []string
	for _, p := range paths {
		patterns = append(patterns, pathPatternsForPath(p)...)
	}
	return patterns
}

func pathPatternsForPath(p path) []string {
	if p.pathType == prefixPathType {
		return buildPatternsForPrefix(p.pathPattern)
	}
	return []string{p.pathPattern}
}

// https://github.com/kubernetes-sigs/aws-load-balancer-controller/pull/1772/files#diff-ab931de004b4ee78d1a27f20f37cc9acaf851ab150094ec8baa1fdbcf5ca67f1R163-R174
func buildPatternsForPrefix(path string) []string {
	if path == "/" {
		return []string{"/*"}
	}

	normalizedPath := strings.TrimSuffix(path, "/")
	return []string{normalizedPath, normalizedPath + "/*"}
}
