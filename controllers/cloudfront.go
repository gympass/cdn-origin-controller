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
	"sort"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
)

func newOrigin(rules []networkingv1.IngressRule, status networkingv1.IngressStatus) cloudfront.Origin {
	host := host(status)
	builder := cloudfront.NewOriginBuilder(host)
	patterns := sortedPaths(rules)
	for _, p := range patterns {
		builder = builder.WithBehavior(p)
	}

	return builder.Build()
}

func host(status networkingv1.IngressStatus) string {
	return status.LoadBalancer.Ingress[0].Hostname
}

func sortedPaths(rules []networkingv1.IngressRule) []string {
	var patterns []string
	for _, r := range rules {
		rulePatterns := pathPatterns(r)
		patterns = append(patterns, rulePatterns...)
	}

	sort.Sort(byLength(patterns))
	return patterns
}

func pathPatterns(rule networkingv1.IngressRule) []string {
	var paths []string
	for _, p := range rule.HTTP.Paths {
		pattern := p.Path
		if *p.PathType == networkingv1.PathTypePrefix {
			pattern = pattern + "*"
		}
		paths = append(paths, pattern)
	}
	return paths
}
