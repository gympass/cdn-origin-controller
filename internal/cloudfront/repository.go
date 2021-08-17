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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
)

const (
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html
	cachingDisabledPolicyID = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html
	allViewerOriginRequestPolicyID = "216adef6-5c7f-47e4-b989-5492eafa07d3"
)

const (
	originSSLProtocolTLSv12 = "TLSv1.2"
	originSSLProtocolTLSv11 = "TLSv1.1"
	originSSLProtocolTLSv1  = "TLSv1"
	originSSLProtocolSSLv3  = "SSLv3"
)

// OriginRepository provides a repository for manipulating CloudFront distributions to match desired configuration
type OriginRepository interface {
	// Save ensures the given origin exists on the CloudFront distribution of given ID
	Save(id string, o Origin) error
}

type repository struct {
	awsClient cloudfrontiface.CloudFrontAPI
}

func NewOriginRepository(awsClient cloudfrontiface.CloudFrontAPI) OriginRepository {
	return &repository{awsClient: awsClient}
}

func (r repository) Save(id string, o Origin) error {
	output, err := r.distributionConfigByID(id)
	if err != nil {
		return fmt.Errorf("getting distribution config: %v", err)
	}

	config := reconcileConfig(*output.DistributionConfig, o)

	updateInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &config,
		IfMatch:            output.ETag,
		Id:                 aws.String(id),
	}
	if _, err = r.awsClient.UpdateDistribution(updateInput); err != nil {
		return fmt.Errorf("updating distribution: %v", err)
	}

	return nil
}

func (r repository) distributionConfigByID(id string) (*awscloudfront.GetDistributionConfigOutput, error) {
	input := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String(id),
	}
	output, err := r.awsClient.GetDistributionConfig(input)

	if err != nil {
		return nil, err
	}
	return output, nil
}

func reconcileConfig(config awscloudfront.DistributionConfig, o Origin) awscloudfront.DistributionConfig {
	config = ensureOriginInConfig(config, newAWSOrigin(o))
	config = ensureBehaviorsInConfig(config, o)
	return config
}

func ensureOriginInConfig(config awscloudfront.DistributionConfig, awsOrigin *awscloudfront.Origin) awscloudfront.DistributionConfig {
	var found bool
	for i, item := range config.Origins.Items {
		if *item.Id == *awsOrigin.Id {
			config.Origins.Items[i] = awsOrigin
			found = true
			break
		}
	}
	if !found {
		config.Origins.Items = append(config.Origins.Items, awsOrigin)
		config.Origins.Quantity = aws.Int64(*config.Origins.Quantity + 1)
	}
	return config
}

func ensureBehaviorsInConfig(config awscloudfront.DistributionConfig, o Origin) awscloudfront.DistributionConfig {
	for _, b := range o.Behaviors {
		config = ensureBehaviorInConfig(config, newCacheBehavior(b, o.Host))
	}
	return config
}

func ensureBehaviorInConfig(config awscloudfront.DistributionConfig, awsBehavior *awscloudfront.CacheBehavior) awscloudfront.DistributionConfig {
	var found bool
	for i, item := range config.CacheBehaviors.Items {
		if *item.PathPattern == *awsBehavior.PathPattern {
			config.CacheBehaviors.Items[i] = awsBehavior
			found = true
			break
		}
	}
	if !found {
		config.CacheBehaviors.Items = append(config.CacheBehaviors.Items, awsBehavior)
		config.CacheBehaviors.Quantity = aws.Int64(*config.CacheBehaviors.Quantity + 1)
	}
	return config
}

func newAWSOrigin(o Origin) *awscloudfront.Origin {
	SSLProtocols := []*string{
		aws.String(originSSLProtocolSSLv3),
		aws.String(originSSLProtocolTLSv1),
		aws.String(originSSLProtocolTLSv11),
		aws.String(originSSLProtocolTLSv12),
	}
	return &awscloudfront.Origin{
		CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
		CustomOriginConfig: &awscloudfront.CustomOriginConfig{
			HTTPPort:               aws.Int64(80),
			HTTPSPort:              aws.Int64(443),
			OriginKeepaliveTimeout: aws.Int64(5),
			OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
			OriginReadTimeout:      aws.Int64(30),
			OriginSslProtocols: &awscloudfront.OriginSslProtocols{
				Items:    SSLProtocols,
				Quantity: aws.Int64(int64(len(SSLProtocols))),
			},
		},
		DomainName: aws.String(o.Host),
		Id:         aws.String(o.Host),
		OriginPath: aws.String(""),
	}
}

func newCacheBehavior(behavior Behavior, host string) *awscloudfront.CacheBehavior {
	return &awscloudfront.CacheBehavior{
		AllowedMethods: &awscloudfront.AllowedMethods{
			Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
			Quantity: aws.Int64(7),
			CachedMethods: &awscloudfront.CachedMethods{
				Items:    aws.StringSlice([]string{"GET", "HEAD"}),
				Quantity: aws.Int64(2),
			},
		},
		CachePolicyId:              aws.String(cachingDisabledPolicyID),
		Compress:                   aws.Bool(true),
		FieldLevelEncryptionId:     aws.String(""),
		LambdaFunctionAssociations: &awscloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
		OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
		PathPattern:                aws.String(behavior.PathPattern),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String(host),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}
}
