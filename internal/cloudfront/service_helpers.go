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

func newDistribution(ingresses []k8s.CDNIngress, group, webACLARN, distARN string, cfg config.Config) (Distribution, error) {
	b := NewDistributionBuilder(
		group,
		cfg,
	)

	for _, ing := range ingresses {
		b = b.WithOrigin(newOrigin(ing, cfg))
		b = b.WithAlternateDomains(ing.AlternateDomainNames)
		b = b.AppendTags(ing.Tags)
		if len(ing.Class.CertificateArn) > 0 && len(cfg.CloudFrontSecurityPolicy) > 0 {
			b = b.WithTLS(ing.Class.CertificateArn, cfg.CloudFrontSecurityPolicy)
		}
	}

	if cfg.CloudFrontEnableIPV6 {
		b = b.WithIPv6()
	}

	if cfg.CloudFrontEnableLogging && len(cfg.CloudFrontS3BucketLog) > 0 {
		b = b.WithLogging(cfg.CloudFrontS3BucketLog, group)
	}

	if len(cfg.CloudFrontCustomTags) > 0 {
		b = b.AppendTags(cfg.CloudFrontCustomTags)
	}

	if len(webACLARN) > 0 {
		b = b.WithWebACL(webACLARN)
	}

	if len(distARN) > 0 {
		b = b.WithARN(distARN)
	}

	return b.Build()
}

func renderDescription(template, group string) string {
	return strings.ReplaceAll(template, "{{group}}", group)
}

func newOrigin(ing k8s.CDNIngress, cfg config.Config) Origin {
	builder := NewOriginBuilder(ing.Group, ing.LoadBalancerHost, ing.OriginAccess, cfg).
		WithViewerFunction(ing.ViewerFnARN).
		WithResponseTimeout(ing.OriginRespTimeout).
		WithRequestPolicy(ing.OriginReqPolicy).
		WithCachePolicy(ing.CachePolicy)

	patterns := pathPatterns(ing.Paths)
	for _, p := range patterns {
		builder = builder.WithBehavior(p)
	}

	return builder.Build()
}

func pathPatterns(paths []k8s.Path) []string {
	var patterns []string
	for _, p := range paths {
		patterns = append(patterns, pathPatternsForPath(p)...)
	}
	return patterns
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
