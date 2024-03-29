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

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

// CallerRefFn is the function that should be called when setting the request's caller reference.
// It should be a unique identifier to prevent the request from being replayed.
// https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_CreateDistribution.html
type CallerRefFn func() string

func newAWSDistributionConfig(d Distribution, callerRef CallerRefFn, cfg config.Config) *cloudfront.DistributionConfig {
	var allCacheBehaviors []*cloudfront.CacheBehavior
	allOrigins := []*cloudfront.Origin{newAWSOrigin(d.DefaultOrigin)}

	for _, o := range d.CustomOrigins {
		allOrigins = append(allOrigins, newAWSOrigin(o))
	}

	for _, b := range d.SortedCustomBehaviors() {
		allCacheBehaviors = append(allCacheBehaviors, newCacheBehavior(b))
	}

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
			},
			CachePolicyId:              aws.String(cfg.CloudFrontDefaultCachingPolicyID),
			Compress:                   aws.Bool(true),
			FieldLevelEncryptionId:     aws.String(""),
			FunctionAssociations:       nil,
			OriginRequestPolicyId:      aws.String(cfg.CloudFrontDefaultCacheRequestPolicyID),
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

	var customOriginConfig *cloudfront.CustomOriginConfig
	var originAccessControlID *string
	var s3OriginConfig *cloudfront.S3OriginConfig

	if o.Access == OriginAccessPublic {
		customOriginConfig = &cloudfront.CustomOriginConfig{
			HTTPPort:               aws.Int64(80),
			HTTPSPort:              aws.Int64(443),
			OriginKeepaliveTimeout: aws.Int64(5),
			OriginProtocolPolicy:   aws.String(cloudfront.OriginProtocolPolicyMatchViewer),
			OriginReadTimeout:      aws.Int64(o.ResponseTimeout),
			OriginSslProtocols: &cloudfront.OriginSslProtocols{
				Items:    SSLProtocols,
				Quantity: aws.Int64(int64(len(SSLProtocols))),
			},
		}
	} else {
		originAccessControlID = &o.OAC.ID
		s3OriginConfig = &cloudfront.S3OriginConfig{
			OriginAccessIdentity: aws.String(""),
		}
	}

	return &cloudfront.Origin{
		CustomHeaders:         newCustomHeaders(o),
		CustomOriginConfig:    customOriginConfig,
		DomainName:            aws.String(o.Host),
		Id:                    aws.String(o.Host),
		OriginAccessControlId: originAccessControlID,
		OriginPath:            aws.String(""),
		S3OriginConfig:        s3OriginConfig,
	}
}

func newCustomHeaders(o Origin) *cloudfront.CustomHeaders {
	var items []*cloudfront.OriginCustomHeader
	for k, v := range o.Headers() {
		items = append(items, &cloudfront.OriginCustomHeader{
			HeaderName:  aws.String(k),
			HeaderValue: aws.String(v),
		})
	}

	return &cloudfront.CustomHeaders{
		Items:    items,
		Quantity: aws.Int64(int64(len(items))),
	}
}

func newCacheBehavior(b Behavior) *cloudfront.CacheBehavior {
	cb := baseCacheBehavior(b)
	var cfFunctions []Function
	var edgeFunctions []Function
	for _, fn := range b.FunctionAssociations {
		switch fn.Type() {
		case k8s.FunctionTypeCloudfront:
			cfFunctions = append(cfFunctions, fn)
		case k8s.FunctionTypeEdge:
			edgeFunctions = append(edgeFunctions, fn)
		}
	}

	cb.SetFunctionAssociations(&cloudfront.FunctionAssociations{
		Items:    newAWSFunctionAssociation(cfFunctions),
		Quantity: aws.Int64(int64(len(cfFunctions))),
	})
	cb.SetLambdaFunctionAssociations(&cloudfront.LambdaFunctionAssociations{
		Items:    newAWSLambdaFunctionAssociation(edgeFunctions),
		Quantity: aws.Int64(int64(len(edgeFunctions))),
	})

	return cb
}

func newAWSFunctionAssociation(functions []Function) []*cloudfront.FunctionAssociation {
	var result []*cloudfront.FunctionAssociation
	for _, fn := range functions {
		result = append(result, &cloudfront.FunctionAssociation{
			EventType:   aws.String(fn.EventType()),
			FunctionARN: aws.String(fn.ARN()),
		})
	}
	return result
}

func newAWSLambdaFunctionAssociation(functions []Function) []*cloudfront.LambdaFunctionAssociation {
	var result []*cloudfront.LambdaFunctionAssociation
	for _, fn := range functions {
		lfa := &cloudfront.LambdaFunctionAssociation{
			EventType:         aws.String(fn.EventType()),
			LambdaFunctionARN: aws.String(fn.ARN()),
		}

		if bfn, ok := fn.(BodyIncluderFunction); ok {
			lfa.SetIncludeBody(bfn.IncludeBody())
		}

		result = append(result, lfa)
	}
	return result
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
