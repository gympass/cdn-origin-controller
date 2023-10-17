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
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

const prefixPathType = string(networkingv1.PathTypePrefix)

func renderDescription(template, group string) string {
	return strings.ReplaceAll(template, "{{group}}", group)
}

func newOrigin(ing k8s.CDNIngress, cfg config.Config, shared k8s.SharedIngressParams) Origin {
	builder := NewOriginBuilder(ing.Group, ing.LoadBalancerHost, ing.OriginAccess, cfg).
		WithResponseTimeout(ing.OriginRespTimeout).
		WithRequestPolicy(ing.OriginReqPolicy).
		WithCachePolicy(ing.CachePolicy)

	for _, p := range shared.PathsFromIngress(ing.NamespacedName) {
		for _, pp := range pathPatternsForPath(p) {
			builder = builder.WithBehavior(pp, NewFunctions(p.FunctionAssociations)...)
		}
	}

	return builder.Build()
}

func pathPatternsForPath(p k8s.Path) []string {
	if p.PathType == prefixPathType {
		return buildPatternsForPrefix(p.PathPattern)
	}
	return []string{p.PathPattern}
}

// https://github.com/kubernetes-sigs/aws-load-balancer-controller/pull/1772/files#diff-ab931de004b4ee78d1a27f20f37cc9acaf851ab150094ec8baa1fdbcf5ca67f1R163-R174
func buildPatternsForPrefix(path string) []string {
	if path == "/" {
		return []string{"/*"}
	}

	normalizedPath := strings.TrimSuffix(path, "/")
	return []string{normalizedPath, normalizedPath + "/*"}
}
