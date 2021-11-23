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
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"k8s.io/apimachinery/pkg/util/wait"
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

// DistributionRepository provides a repository for manipulating CloudFront distributions to match desired configuration
type DistributionRepository interface {
	// Create creates the given Distribution on CloudFront
	Create(Distribution) (Distribution, error)
	// Sync ensures the given Distribution is correctly configured on CloudFront
	Sync(Distribution) error
	// Delete deletes the Distribution at AWS
	Delete(distribution Distribution) error
}

// CallerRefFn is the function that should be called when setting the request's caller reference.
// It should be a unique identifier to prevent the request from being replayed.
// https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_CreateDistribution.html
type CallerRefFn func() string

type repository struct {
	awsClient   cloudfrontiface.CloudFrontAPI
	callerRef   CallerRefFn
	waitTimeout time.Duration
}

// NewDistributionRepository creates a new AWS CloudFront DistributionRepository
func NewDistributionRepository(awsClient cloudfrontiface.CloudFrontAPI, callerRefFn CallerRefFn, waitTimeout time.Duration) DistributionRepository {
	return &repository{awsClient: awsClient, callerRef: callerRefFn, waitTimeout: waitTimeout}
}

func (r repository) Create(d Distribution) (Distribution, error) {
	config := r.newAWSDistributionConfig(d)
	createInput := &awscloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &awscloudfront.DistributionConfigWithTags{
			DistributionConfig: config,
			Tags:               r.distributionTags(d),
		},
	}
	output, err := r.awsClient.CreateDistributionWithTags(createInput)
	if err != nil {
		return Distribution{}, fmt.Errorf("creating distribution: %v", err)
	}

	d.ID = *output.Distribution.Id
	d.Address = *output.Distribution.DomainName
	d.ARN = *output.Distribution.ARN
	return d, nil
}

func (r repository) Sync(d Distribution) error {
	config := r.newAWSDistributionConfig(d)
	output, err := r.distributionConfigByID(d.ID)
	if err != nil {
		return fmt.Errorf("getting distribution config: %v", err)
	}

	config.SetCallerReference(*output.DistributionConfig.CallerReference)
	config.SetDefaultRootObject(*output.DistributionConfig.DefaultRootObject)
	config.SetCustomErrorResponses(output.DistributionConfig.CustomErrorResponses)
	config.SetRestrictions(output.DistributionConfig.Restrictions)

	updateInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: config,
		IfMatch:            output.ETag,
		Id:                 aws.String(d.ID),
	}

	if _, err = r.awsClient.UpdateDistribution(updateInput); err != nil {
		return fmt.Errorf("updating distribution: %v", err)
	}

	tagsInput := &awscloudfront.TagResourceInput{
		Resource: aws.String(d.ARN),
		Tags:     r.distributionTags(d),
	}

	if _, err = r.awsClient.TagResource(tagsInput); err != nil {
		return fmt.Errorf("updating tags: %v", err)
	}

	return nil
}

func (r repository) Delete(d Distribution) error {
	output, err := r.distributionConfigByID(d.ID)
	if isNoSuchDistributionErr(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("getting distribution config: %v", err)
	}

	if *output.DistributionConfig.Enabled {
		err = r.disableDist(output.DistributionConfig, d.ID, *output.ETag)
		if isNoSuchDistributionErr(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("disabling distribution: %v", err)
		}
	}

	eTag, err := r.waitUntilDeployed(d.ID)
	if isNoSuchDistributionErr(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("waiting for distribution to be in deployed status: %w", err)
	}

	input := &awscloudfront.DeleteDistributionInput{
		Id:      aws.String(d.ID),
		IfMatch: eTag,
	}
	_, err = r.awsClient.DeleteDistribution(input)
	if isNoSuchDistributionErr(err) {
		err = nil
	}
	return err
}

func (r repository) distributionTags(d Distribution) *awscloudfront.Tags {
	var awsTags awscloudfront.Tags
	for k, v := range d.Tags {
		awsTags.Items = append(awsTags.Items, &awscloudfront.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	// map iteration is non-deterministic, so we sort the tags to make this deterministic and testable
	sort.Sort(byKey(awsTags.Items))
	return &awsTags
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

func (r repository) disableDist(config *awscloudfront.DistributionConfig, id, eTag string) error {
	config.Enabled = aws.Bool(false)
	updateInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: config,
		IfMatch:            aws.String(eTag),
		Id:                 aws.String(id),
	}

	_, err := r.awsClient.UpdateDistribution(updateInput)
	return err
}

const cfDeployedStatus = "Deployed"

func (r repository) waitUntilDeployed(id string) (*string, error) {
	var eTag *string
	condition := func() (done bool, err error) {
		out, err := r.distributionByID(id)
		if err != nil {
			if isNoSuchDistributionErr(err) {
				return false, err
			}
			return false, nil
		}
		eTag = out.ETag
		return *out.Distribution.Status == cfDeployedStatus, nil
	}

	interval := time.Second * 10
	err := wait.PollImmediate(interval, r.waitTimeout, condition)
	if err != nil {
		return nil, err
	}
	return eTag, nil
}

func (r repository) distributionByID(id string) (*awscloudfront.GetDistributionOutput, error) {
	input := &awscloudfront.GetDistributionInput{
		Id: aws.String(id),
	}
	return r.awsClient.GetDistribution(input)
}

func (r repository) newAWSDistributionConfig(d Distribution) *awscloudfront.DistributionConfig {
	var allCacheBehaviors []*awscloudfront.CacheBehavior
	allOrigins := []*awscloudfront.Origin{newAWSOrigin(d.DefaultOrigin)}

	for _, o := range d.CustomOrigins {
		allOrigins = append(allOrigins, newAWSOrigin(o))
		for _, b := range o.Behaviors {
			allCacheBehaviors = append(allCacheBehaviors, newCacheBehavior(b, o.Host))
		}
	}

	sort.Sort(byDescendingPathLength(allCacheBehaviors))

	allOrigins = removeDuplicates(allOrigins)

	config := &awscloudfront.DistributionConfig{
		Aliases: &awscloudfront.Aliases{
			Items:    aws.StringSlice(d.AlternateDomains),
			Quantity: aws.Int64(int64(len(d.AlternateDomains))),
		},
		CacheBehaviors: &awscloudfront.CacheBehaviors{
			Items:    allCacheBehaviors,
			Quantity: aws.Int64(int64(len(allCacheBehaviors))),
		},
		CallerReference:      aws.String(r.callerRef()),
		Comment:              aws.String(d.Description),
		CustomErrorResponses: nil,
		DefaultCacheBehavior: &awscloudfront.DefaultCacheBehavior{
			AllowedMethods: &awscloudfront.AllowedMethods{
				Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
				Quantity: aws.Int64(7),
				CachedMethods: &awscloudfront.CachedMethods{
					Items:    aws.StringSlice([]string{"GET", "HEAD"}),
					Quantity: aws.Int64(2),
				},
			}, CachePolicyId: aws.String(cachingDisabledPolicyID),
			Compress:                   aws.Bool(true),
			FieldLevelEncryptionId:     aws.String(""),
			FunctionAssociations:       nil,
			OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
			LambdaFunctionAssociations: &awscloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
			RealtimeLogConfigArn:       nil,
			SmoothStreaming:            aws.Bool(false),
			TargetOriginId:             aws.String(d.DefaultOrigin.Host),
			TrustedKeyGroups:           nil,
			TrustedSigners:             nil,
			ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
		},
		Origins: &awscloudfront.Origins{
			Items:    allOrigins,
			Quantity: aws.Int64(int64(len(allOrigins))),
		},
		DefaultRootObject: nil,
		Enabled:           aws.Bool(true),
		HttpVersion:       aws.String(awscloudfront.HttpVersionHttp2),
		IsIPV6Enabled:     aws.Bool(d.IPv6Enabled),
		Logging:           nil,
		OriginGroups:      nil,
		PriceClass:        aws.String(d.PriceClass),
		Restrictions:      nil,
		ViewerCertificate: nil,
		WebACLId:          aws.String(d.WebACLID),
	}

	if d.TLS.Enabled {
		config.ViewerCertificate = &awscloudfront.ViewerCertificate{
			ACMCertificateArn:      aws.String(d.TLS.CertARN),
			MinimumProtocolVersion: aws.String(d.TLS.SecurityPolicyID),
			SSLSupportMethod:       aws.String(awscloudfront.SSLSupportMethodSniOnly),
		}
	}
	if d.Logging.Enabled {
		config.Logging = &awscloudfront.LoggingConfig{
			Enabled:        aws.Bool(true),
			Bucket:         aws.String(d.Logging.BucketAddress),
			Prefix:         aws.String(d.Logging.Prefix),
			IncludeCookies: aws.Bool(false),
		}
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
			OriginReadTimeout:      aws.Int64(o.ResponseTimeout),
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
	cb := baseCacheBehavior(behavior.PathPattern, host)
	if len(behavior.ViewerFnARN) > 0 {
		addViewerFunctionAssociation(cb, behavior.ViewerFnARN)
	}
	return cb
}

func addViewerFunctionAssociation(cb *awscloudfront.CacheBehavior, functionARN string) {
	cb.FunctionAssociations = &awscloudfront.FunctionAssociations{
		Items: []*awscloudfront.FunctionAssociation{
			{
				FunctionARN: aws.String(functionARN),
				EventType:   aws.String(awscloudfront.EventTypeViewerRequest),
			},
		},
		Quantity: aws.Int64(1),
	}
}

func baseCacheBehavior(pathPattern, host string) *awscloudfront.CacheBehavior {
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
		PathPattern:                aws.String(pathPattern),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String(host),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}
}

func removeDuplicates(origins []*awscloudfront.Origin) []*awscloudfront.Origin {
	var result []*awscloudfront.Origin
	foundSet := make(map[string]bool)
	for _, origin := range origins {
		if !foundSet[*origin.DomainName] {
			foundSet[*origin.DomainName] = true
			result = append(result, origin)
		}
	}
	return result
}

func isNoSuchDistributionErr(err error) bool {
	var aerr awserr.Error
	if ok := errors.As(err, &aerr); !ok {
		return false
	}
	return aerr.Code() == awscloudfront.ErrCodeNoSuchDistribution
}
