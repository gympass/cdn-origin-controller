// Copyright (c) 2022 GPBR Participacoes LTDA.
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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

// CallerRefFn is the function that should be called when setting the request's caller reference.
// It should be a unique identifier to prevent the request from being replayed.
// https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_CreateDistribution.html
type CallerRefFn func() string

func newAWSDistributionConfig(d Distribution, callerRef CallerRefFn) *cloudfront.DistributionConfig {
	var allCacheBehaviors []*cloudfront.CacheBehavior
	allOrigins := []*cloudfront.Origin{newAWSOrigin(d.DefaultOrigin)}

	for _, o := range d.CustomOrigins {
		allOrigins = append(allOrigins, newAWSOrigin(o))
	}

	for _, b := range d.CustomBehaviors() {
		allCacheBehaviors = append(allCacheBehaviors, newCacheBehavior(b))
	}

	allOrigins = removeDuplicates(allOrigins)

	config := &cloudfront.DistributionConfig{
		Aliases: &cloudfront.Aliases{
			Items:    aws.StringSlice(d.AlternateDomains),
			Quantity: aws.Int64(int64(len(d.AlternateDomains))),
		},
		CacheBehaviors: &cloudfront.CacheBehaviors{
			Items:    allCacheBehaviors,
			Quantity: aws.Int64(int64(len(allCacheBehaviors))),
		},
		CallerReference:      aws.String(callerRef()),
		Comment:              aws.String(d.Description),
		CustomErrorResponses: nil,
		DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
			AllowedMethods: &cloudfront.AllowedMethods{
				Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
				Quantity: aws.Int64(7),
				CachedMethods: &cloudfront.CachedMethods{
					Items:    aws.StringSlice([]string{"GET", "HEAD"}),
					Quantity: aws.Int64(2),
				},
			}, CachePolicyId: aws.String(cachingDisabledPolicyID),
			Compress:                   aws.Bool(true),
			FieldLevelEncryptionId:     aws.String(""),
			FunctionAssociations:       nil,
			OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
			LambdaFunctionAssociations: &cloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
			RealtimeLogConfigArn:       nil,
			SmoothStreaming:            aws.Bool(false),
			TargetOriginId:             aws.String(d.DefaultOrigin.Host),
			TrustedKeyGroups:           nil,
			TrustedSigners:             nil,
			ViewerProtocolPolicy:       aws.String(cloudfront.ViewerProtocolPolicyRedirectToHttps),
		},
		Origins: &cloudfront.Origins{
			Items:    allOrigins,
			Quantity: aws.Int64(int64(len(allOrigins))),
		},
		DefaultRootObject: nil,
		Enabled:           aws.Bool(true),
		HttpVersion:       aws.String(cloudfront.HttpVersionHttp2),
		IsIPV6Enabled:     aws.Bool(d.IPv6Enabled),
		Logging: &cloudfront.LoggingConfig{
			Enabled:        aws.Bool(false),
			Bucket:         aws.String(""),
			Prefix:         aws.String(""),
			IncludeCookies: aws.Bool(false),
		},
		OriginGroups:      nil,
		PriceClass:        aws.String(d.PriceClass),
		Restrictions:      nil,
		ViewerCertificate: nil,
		WebACLId:          aws.String(d.WebACLID),
	}

	if d.TLS.Enabled {
		config.ViewerCertificate = &cloudfront.ViewerCertificate{
			ACMCertificateArn:      aws.String(d.TLS.CertARN),
			MinimumProtocolVersion: aws.String(d.TLS.SecurityPolicyID),
			SSLSupportMethod:       aws.String(cloudfront.SSLSupportMethodSniOnly),
		}
	}
	if d.Logging.Enabled {
		config.Logging = &cloudfront.LoggingConfig{
			Enabled:        aws.Bool(true),
			Bucket:         aws.String(d.Logging.BucketAddress),
			Prefix:         aws.String(d.Logging.Prefix),
			IncludeCookies: aws.Bool(false),
		}
	}

	return config
}

func newAWSOrigin(o Origin) *cloudfront.Origin {
	SSLProtocols := []*string{
		aws.String(originSSLProtocolSSLv3),
		aws.String(originSSLProtocolTLSv1),
		aws.String(originSSLProtocolTLSv11),
		aws.String(originSSLProtocolTLSv12),
	}

	return &cloudfront.Origin{
		CustomHeaders: &cloudfront.CustomHeaders{Quantity: aws.Int64(0)},
		CustomOriginConfig: &cloudfront.CustomOriginConfig{
			HTTPPort:               aws.Int64(80),
			HTTPSPort:              aws.Int64(443),
			OriginKeepaliveTimeout: aws.Int64(5),
			OriginProtocolPolicy:   aws.String(cloudfront.OriginProtocolPolicyMatchViewer),
			OriginReadTimeout:      aws.Int64(o.ResponseTimeout),
			OriginSslProtocols: &cloudfront.OriginSslProtocols{
				Items:    SSLProtocols,
				Quantity: aws.Int64(int64(len(SSLProtocols))),
			},
		},
		DomainName: aws.String(o.Host),
		Id:         aws.String(o.Host),
		OriginPath: aws.String(""),
	}
}

func newCacheBehavior(b Behavior) *cloudfront.CacheBehavior {
	cb := baseCacheBehavior(b)
	if len(b.ViewerFnARN) > 0 {
		addViewerFunctionAssociation(cb, b.ViewerFnARN)
	}
	return cb
}

func addViewerFunctionAssociation(cb *cloudfront.CacheBehavior, functionARN string) {
	cb.FunctionAssociations = &cloudfront.FunctionAssociations{
		Items: []*cloudfront.FunctionAssociation{
			{
				FunctionARN: aws.String(functionARN),
				EventType:   aws.String(cloudfront.EventTypeViewerRequest),
			},
		},
		Quantity: aws.Int64(1),
	}
}

func baseCacheBehavior(b Behavior) *cloudfront.CacheBehavior {
	cb := &cloudfront.CacheBehavior{
		AllowedMethods: &cloudfront.AllowedMethods{
			Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
			Quantity: aws.Int64(7),
			CachedMethods: &cloudfront.CachedMethods{
				Items:    aws.StringSlice([]string{"GET", "HEAD"}),
				Quantity: aws.Int64(2),
			},
		},
		CachePolicyId:              aws.String(b.CachePolicy),
		Compress:                   aws.Bool(true),
		FieldLevelEncryptionId:     aws.String(""),
		LambdaFunctionAssociations: &cloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
		OriginRequestPolicyId:      aws.String(b.RequestPolicy),
		PathPattern:                aws.String(b.PathPattern),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String(b.OriginHost),
		ViewerProtocolPolicy:       aws.String(cloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	if b.RequestPolicy == "None" {
		cb.OriginRequestPolicyId = nil
	}

	return cb
}

func removeDuplicates(origins []*cloudfront.Origin) []*cloudfront.Origin {
	var result []*cloudfront.Origin
	foundSet := make(map[string]bool)
	for _, origin := range origins {
		if !foundSet[*origin.DomainName] {
			foundSet[*origin.DomainName] = true
			result = append(result, origin)
		}
	}
	return result
}
