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

type Repository interface {
	Update(id string, o Origin) error
}

type repository struct {
	awsClient cloudfrontiface.CloudFrontAPI
}

func NewRepository(awsClient cloudfrontiface.CloudFrontAPI) Repository {
	return &repository{awsClient: awsClient}
}

func (r repository) Update(id string, o Origin) error {
	output, err := r.distributionConfigByID(id)

	if err != nil {
		return fmt.Errorf("getting distribution config: %v", err)
	}

	var found bool

	for _, item := range output.DistributionConfig.Origins.Items {
		if item.Id == &o.Host {
			found = true
			break
		}
	}

	if !found {
		cfOrigin := &awscloudfront.Origin{
			Id:         aws.String(o.Host),
			DomainName: aws.String(o.Host),
		}
		output.DistributionConfig.Origins.Items = append(output.DistributionConfig.Origins.Items, cfOrigin)
	}

	found = false

	for _, behavior := range o.Behaviors {
		for _, item := range output.DistributionConfig.CacheBehaviors.Items {
			if item.PathPattern == &behavior.PathPattern {
				item.TargetOriginId = aws.String(o.Host)
				found = true
				break
			}
		}

		if !found {
			cfBehavior := awscloudfront.CacheBehavior{
				PathPattern:          aws.String(behavior.PathPattern),
				TargetOriginId:       aws.String(o.Host),
				Compress:             aws.Bool(true),
				ViewerProtocolPolicy: aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
				AllowedMethods: &awscloudfront.AllowedMethods{
					Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
					Quantity: aws.Int64(7),
				},
				CachePolicyId:         aws.String(cachingDisabledPolicyID),
				OriginRequestPolicyId: aws.String(allViewerOriginRequestPolicyID),
			}
			output.DistributionConfig.CacheBehaviors.Items = append(output.DistributionConfig.CacheBehaviors.Items, &cfBehavior)
		}
	}

	// ViewerProtocolPolicyRedirectToHttps
	r.awsClient.UpdateDistribution(&awscloudfront.UpdateDistributionInput{
		DistributionConfig: output.DistributionConfig,
		IfMatch:            output.ETag,
	})
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
