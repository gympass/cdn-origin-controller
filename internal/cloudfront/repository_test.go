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
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	sslProtocols = []*string{
		aws.String("SSLv3"),
		aws.String("TLSv1"),
		aws.String("TLSv1.1"),
		aws.String("TLSv1.2"),
	}
	defaultOrigin = &awscloudfront.Origin{
		CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
		CustomOriginConfig: &awscloudfront.CustomOriginConfig{
			HTTPPort:               aws.Int64(80),
			HTTPSPort:              aws.Int64(443),
			OriginKeepaliveTimeout: aws.Int64(5),
			OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
			OriginReadTimeout:      aws.Int64(30),
			OriginSslProtocols: &awscloudfront.OriginSslProtocols{
				Items:    sslProtocols,
				Quantity: aws.Int64(int64(len(sslProtocols))),
			},
		},
		DomainName: aws.String("default.origin"),
		Id:         aws.String("default.origin"),
		OriginPath: aws.String(""),
	}

	expectedDefaultCacheBehavior = &awscloudfront.DefaultCacheBehavior{
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
		TargetOriginId:             aws.String("default.origin"),
		TrustedKeyGroups:           nil,
		TrustedSigners:             nil,
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	testCallerRefFn = func() string { return "test caller ref" }
)

type awsClientMock struct {
	mock.Mock
	cloudfrontiface.CloudFrontAPI
	expectedGetDistributionConfigOutput      *awscloudfront.GetDistributionConfigOutput
	expectedUpdateDistributionOutput         *awscloudfront.UpdateDistributionOutput
	expectedCreateDistributionWithTagsOutput *awscloudfront.CreateDistributionWithTagsOutput
	expectedTagResourceOutput                *awscloudfront.TagResourceOutput
	expectedGetDistributionOutput            *awscloudfront.GetDistributionOutput
}

func (c *awsClientMock) GetDistributionConfig(in *awscloudfront.GetDistributionConfigInput) (*awscloudfront.GetDistributionConfigOutput, error) {
	args := c.Called(in)
	return c.expectedGetDistributionConfigOutput, args.Error(0)
}

func (c *awsClientMock) UpdateDistribution(in *awscloudfront.UpdateDistributionInput) (*awscloudfront.UpdateDistributionOutput, error) {
	args := c.Called(in)
	return c.expectedUpdateDistributionOutput, args.Error(0)
}
func (c *awsClientMock) GetDistribution(in *awscloudfront.GetDistributionInput) (*awscloudfront.GetDistributionOutput, error) {
	args := c.Called(in)
	return c.expectedGetDistributionOutput, args.Error(0)
}

func (c *awsClientMock) CreateDistributionWithTags(in *awscloudfront.CreateDistributionWithTagsInput) (*awscloudfront.CreateDistributionWithTagsOutput, error) {
	args := c.Called(in)
	return c.expectedCreateDistributionWithTagsOutput, args.Error(0)
}

func (c *awsClientMock) TagResource(in *awscloudfront.TagResourceInput) (*awscloudfront.TagResourceOutput, error) {
	args := c.Called(in)
	return c.expectedTagResourceOutput, args.Error(0)
}

func (c *awsClientMock) DeleteDistribution(in *awscloudfront.DeleteDistributionInput) (*awscloudfront.DeleteDistributionOutput, error) {
	args := c.Called(in)
	return nil, args.Error(0)
}

func TestRunDistributionRepositoryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DistributionRepositoryTestSuite{})
}

type DistributionRepositoryTestSuite struct {
	suite.Suite
}

func (s *DistributionRepositoryTestSuite) TestCreate_Success() {
	awsClient := &awsClientMock{
		expectedCreateDistributionWithTagsOutput: &awscloudfront.CreateDistributionWithTagsOutput{
			Distribution: &awscloudfront.Distribution{
				Id:         aws.String("L2FB5NP10VU7KL"),
				ARN:        aws.String("arn:aws:cloudfront::123456789012:distribution/L2FB5NP10VU7KL"),
				DomainName: aws.String("aoiweoiwe39d.cloudfront.net"),
			},
		},
	}

	expectedCreateInput := &awscloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &awscloudfront.DistributionConfigWithTags{
			Tags: &awscloudfront.Tags{
				Items: []*awscloudfront.Tag{
					{
						Key:   aws.String("cdn-origin-controller.gympass.com/cdn.group"),
						Value: aws.String("test group"),
					},
					{
						Key:   aws.String("cdn-origin-controller.gympass.com/owned"),
						Value: aws.String("true"),
					},
					{
						Key:   aws.String("foo"),
						Value: aws.String("bar"),
					},
				},
			},
			DistributionConfig: &awscloudfront.DistributionConfig{
				Aliases: &awscloudfront.Aliases{
					Items:    aws.StringSlice([]string{"test.alias.1", "test.alias.2"}),
					Quantity: aws.Int64(2),
				},
				CallerReference: aws.String(testCallerRefFn()),
				CacheBehaviors: &awscloudfront.CacheBehaviors{
					Quantity: aws.Int64(0),
				},
				Comment: aws.String("test description"),
				DefaultCacheBehavior: &awscloudfront.DefaultCacheBehavior{
					Compress:                   aws.Bool(true),
					FieldLevelEncryptionId:     aws.String(""),
					SmoothStreaming:            aws.Bool(false),
					LambdaFunctionAssociations: &awscloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
					CachePolicyId:              aws.String(cachingDisabledPolicyID),
					OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
					TargetOriginId:             aws.String("default.origin"),
					ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
					AllowedMethods: &awscloudfront.AllowedMethods{
						Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
						Quantity: aws.Int64(7),
						CachedMethods: &awscloudfront.CachedMethods{
							Items:    aws.StringSlice([]string{"GET", "HEAD"}),
							Quantity: aws.Int64(2),
						},
					},
				},
				Enabled:       aws.Bool(true),
				HttpVersion:   aws.String(awscloudfront.HttpVersionHttp2),
				IsIPV6Enabled: aws.Bool(true),
				Logging: &awscloudfront.LoggingConfig{
					Enabled:        aws.Bool(true),
					Bucket:         aws.String("test s3"),
					Prefix:         aws.String("test prefix"),
					IncludeCookies: aws.Bool(false),
				},
				Origins: &awscloudfront.Origins{
					Items: []*awscloudfront.Origin{
						defaultOrigin,
						{
							CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
							CustomOriginConfig: &awscloudfront.CustomOriginConfig{
								HTTPPort:               aws.Int64(80),
								HTTPSPort:              aws.Int64(443),
								OriginKeepaliveTimeout: aws.Int64(5),
								OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
								OriginReadTimeout:      aws.Int64(30),
								OriginSslProtocols: &awscloudfront.OriginSslProtocols{
									Items:    sslProtocols,
									Quantity: aws.Int64(int64(len(sslProtocols))),
								},
							},
							DomainName: aws.String("origin"),
							Id:         aws.String("origin"),
							OriginPath: aws.String(""),
						},
					},
					Quantity: aws.Int64(2),
				},
				PriceClass: aws.String(awscloudfront.PriceClassPriceClass100),
				ViewerCertificate: &awscloudfront.ViewerCertificate{
					ACMCertificateArn:      aws.String("test:cert:arn"),
					MinimumProtocolVersion: aws.String("test security policy"),
					SSLSupportMethod:       aws.String(awscloudfront.SSLSupportMethodSniOnly),
				},
				WebACLId: aws.String("test web acl"),
			},
		},
	}

	var noError error
	awsClient.On("CreateDistributionWithTags", expectedCreateInput).Return(noError).Once()

	distribution, err := NewDistributionBuilder(
		"default.origin",
		"test description",
		awscloudfront.PriceClassPriceClass100,
		"test group",
	).
		WithOrigin(Origin{Host: "origin", ResponseTimeout: 30}).
		WithAlternateDomains([]string{"test.alias.1", "test.alias.2"}).
		WithWebACL("test web acl").
		WithTags(map[string]string{"foo": "bar"}).
		WithLogging("test s3", "test prefix").
		WithTLS("test:cert:arn", "test security policy").
		WithIPv6().
		Build()
	s.NoError(err)

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	dist, err := repo.Create(distribution)
	s.Equal(dist.ID, "L2FB5NP10VU7KL")
	s.Equal(dist.ARN, "arn:aws:cloudfront::123456789012:distribution/L2FB5NP10VU7KL")
	s.Equal(dist.Address, "aoiweoiwe39d.cloudfront.net")
	s.NoError(err)
}

func (s *DistributionRepositoryTestSuite) TestCreate_ErrorWhenCreatingDistribution() {
	awsClient := &awsClientMock{}
	awsClient.On("CreateDistributionWithTags", mock.Anything).Return(errors.New("mock err")).Once()

	distribution := Distribution{
		ID: "mock id",
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
			},
		},
	}

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	dist, err := repo.Create(distribution)
	s.Equal(Distribution{}, dist)
	s.Error(err)
}

func (s *DistributionRepositoryTestSuite) TestSync_CantFetchDistribution() {
	awsClient := &awsClientMock{}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(errors.New("mock err")).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.Error(repo.Sync(Distribution{}))
}

func (s *DistributionRepositoryTestSuite) TestSync_CantUpdateDistribution() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", mock.Anything).Return(errors.New("mock err")).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.Error(repo.Sync(Distribution{}))
}

func (s *DistributionRepositoryTestSuite) TestSync_CantSaveTags() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	awsClient.On("TagResource", mock.Anything).Return(errors.New("mock err")).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.Error(repo.Sync(Distribution{}))
}

func (s *DistributionRepositoryTestSuite) TestSync_OriginDoesNotExistYet() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
			Comment:              aws.String(""),
			HttpVersion:          aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled:        aws.Bool(false),
			WebACLId:             aws.String(""),
			PriceClass:           aws.String(""),
			Origins: &awscloudfront.Origins{
				Items: []*awscloudfront.Origin{
					defaultOrigin,
					{
						CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
						CustomOriginConfig: &awscloudfront.CustomOriginConfig{
							HTTPPort:               aws.Int64(80),
							HTTPSPort:              aws.Int64(443),
							OriginKeepaliveTimeout: aws.Int64(5),
							OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
							OriginReadTimeout:      aws.Int64(30),
							OriginSslProtocols: &awscloudfront.OriginSslProtocols{
								Items:    sslProtocols,
								Quantity: aws.Int64(int64(len(sslProtocols))),
							},
						},
						DomainName: aws.String("origin"),
						Id:         aws.String("origin"),
						OriginPath: aws.String(""),
					},
				},
				Quantity: aws.Int64(2),
			},
			DefaultCacheBehavior: expectedDefaultCacheBehavior,
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("TagResource", mock.Anything).Return(noError).Once()

	distribution := Distribution{
		ID: "mock id",
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
			},
		},
	}

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestSync_OriginAlreadyExists() {
	someIncorrectOrigin := &awscloudfront.Origin{Id: aws.String("origin"), DomainName: aws.String("incorrect domain name")}

	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              &awscloudfront.Origins{Items: []*awscloudfront.Origin{someIncorrectOrigin}, Quantity: aws.Int64(1)},
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
			Comment:              aws.String(""),
			HttpVersion:          aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled:        aws.Bool(false),
			WebACLId:             aws.String(""),
			PriceClass:           aws.String(""),
			Origins: &awscloudfront.Origins{
				Items: []*awscloudfront.Origin{
					defaultOrigin,
					{
						CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
						CustomOriginConfig: &awscloudfront.CustomOriginConfig{
							HTTPPort:               aws.Int64(80),
							HTTPSPort:              aws.Int64(443),
							OriginKeepaliveTimeout: aws.Int64(5),
							OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
							OriginReadTimeout:      aws.Int64(30),
							OriginSslProtocols: &awscloudfront.OriginSslProtocols{
								Items:    sslProtocols,
								Quantity: aws.Int64(int64(len(sslProtocols))),
							},
						},
						DomainName: aws.String("origin"),
						Id:         aws.String("origin"),
						OriginPath: aws.String(""),
					},
				},
				Quantity: aws.Int64(2),
			},
			DefaultCacheBehavior: expectedDefaultCacheBehavior,
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("TagResource", mock.Anything).Return(noError).Once()

	distribution := Distribution{
		ID: "mock id",
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
			},
		},
	}

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestSync_BehaviorDoesNotExistYet() {

	lowerPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{
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
		OriginRequestPolicyId:      aws.String("policy"),
		PathPattern:                aws.String("/low/precedence/path"),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String("origin"),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	higherPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{
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
		OriginRequestPolicyId:      aws.String("policy"),
		PathPattern:                aws.String("/very/high/precedence/path/very/lengthy/indeed"),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String("origin"),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(2), Items: []*awscloudfront.CacheBehavior{higherPrecedenceExistingBehavior, lowerPrecedenceExistingBehavior}},
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	expectedNewCacheBehavior := &awscloudfront.CacheBehavior{
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
		OriginRequestPolicyId:      aws.String("policy"),
		PathPattern:                aws.String("/mid-sized/path/with/medium/precedence"),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String("origin"),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
			Comment:              aws.String(""),
			HttpVersion:          aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled:        aws.Bool(false),
			WebACLId:             aws.String(""),
			PriceClass:           aws.String(""),
			Origins: &awscloudfront.Origins{
				Items: []*awscloudfront.Origin{
					defaultOrigin,
					{
						CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
						CustomOriginConfig: &awscloudfront.CustomOriginConfig{
							HTTPPort:               aws.Int64(80),
							HTTPSPort:              aws.Int64(443),
							OriginKeepaliveTimeout: aws.Int64(5),
							OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
							OriginReadTimeout:      aws.Int64(30),
							OriginSslProtocols: &awscloudfront.OriginSslProtocols{
								Items:    sslProtocols,
								Quantity: aws.Int64(int64(len(sslProtocols))),
							},
						},
						DomainName: aws.String("origin"),
						Id:         aws.String("origin"),
						OriginPath: aws.String(""),
					},
				},
				Quantity: aws.Int64(2),
			},
			DefaultCacheBehavior: expectedDefaultCacheBehavior,
			CacheBehaviors: &awscloudfront.CacheBehaviors{
				Items: []*awscloudfront.CacheBehavior{
					higherPrecedenceExistingBehavior,
					expectedNewCacheBehavior,
					lowerPrecedenceExistingBehavior,
				},
				Quantity: aws.Int64(3),
			},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("TagResource", mock.Anything).Return(noError).Once()

	distribution := Distribution{
		ID: "mock id",
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
				Behaviors: []Behavior{
					{PathPattern: "/mid-sized/path/with/medium/precedence", RequestPolicy: "policy"},
					{PathPattern: "/low/precedence/path", RequestPolicy: "policy"},
					{PathPattern: "/very/high/precedence/path/very/lengthy/indeed", RequestPolicy: "policy"},
				},
			},
		},
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
	}

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestSync_BehaviorAlreadyExists() {
	existingOrigins := &awscloudfront.Origins{
		Items: []*awscloudfront.Origin{
			defaultOrigin,
			{
				CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
				CustomOriginConfig: &awscloudfront.CustomOriginConfig{
					HTTPPort:               aws.Int64(80),
					HTTPSPort:              aws.Int64(443),
					OriginKeepaliveTimeout: aws.Int64(5),
					OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
					OriginReadTimeout:      aws.Int64(30),
					OriginSslProtocols: &awscloudfront.OriginSslProtocols{
						Items:    sslProtocols,
						Quantity: aws.Int64(int64(len(sslProtocols))),
					},
				},
				DomainName: aws.String("origin"),
				Id:         aws.String("origin"),
				OriginPath: aws.String(""),
			},
		},
		Quantity: aws.Int64(2),
	}

	someIncorrectBehavior := &awscloudfront.CacheBehavior{PathPattern: aws.String("/*"), SmoothStreaming: aws.Bool(true)}
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              existingOrigins,
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Items: []*awscloudfront.CacheBehavior{someIncorrectBehavior}, Quantity: aws.Int64(1)},
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
			Enabled:              aws.Bool(true),
			Comment:              aws.String(""),
			HttpVersion:          aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled:        aws.Bool(false),
			WebACLId:             aws.String(""),
			PriceClass:           aws.String(""),
			Origins:              existingOrigins,
			DefaultCacheBehavior: expectedDefaultCacheBehavior,
			CacheBehaviors: &awscloudfront.CacheBehaviors{
				Items: []*awscloudfront.CacheBehavior{
					{
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
						OriginRequestPolicyId:      aws.String("policy"),
						PathPattern:                aws.String("/*"),
						SmoothStreaming:            aws.Bool(false),
						TargetOriginId:             aws.String("origin"),
						ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
					},
				},
				Quantity: aws.Int64(1),
			},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("TagResource", mock.Anything).Return(noError).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)

	distribution := Distribution{
		ID: "mock id",
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
				Behaviors:       []Behavior{{PathPattern: "/*", RequestPolicy: "policy"}},
			},
		},
	}
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestSync_WithViewerFunction() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String("foo"),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	expectedDefaultCacheBehavior := &awscloudfront.DefaultCacheBehavior{
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
		TargetOriginId:             aws.String("default.origin"),
		TrustedKeyGroups:           nil,
		TrustedSigners:             nil,
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	expectedNewCacheBehavior := &awscloudfront.CacheBehavior{
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
		OriginRequestPolicyId:      aws.String("policy"),
		PathPattern:                aws.String("/foo"),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String("origin"),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
		FunctionAssociations: &awscloudfront.FunctionAssociations{
			Items: []*awscloudfront.FunctionAssociation{
				{
					EventType:   aws.String(awscloudfront.EventTypeViewerRequest),
					FunctionARN: aws.String("some-arn"),
				},
			},
			Quantity: aws.Int64(1),
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Enabled:              aws.Bool(true),
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
			Comment:              aws.String(""),
			HttpVersion:          aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled:        aws.Bool(false),
			WebACLId:             aws.String(""),
			PriceClass:           aws.String(""),
			Origins: &awscloudfront.Origins{
				Items: []*awscloudfront.Origin{
					defaultOrigin,
					{
						CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
						CustomOriginConfig: &awscloudfront.CustomOriginConfig{
							HTTPPort:               aws.Int64(80),
							HTTPSPort:              aws.Int64(443),
							OriginKeepaliveTimeout: aws.Int64(5),
							OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
							OriginReadTimeout:      aws.Int64(30),
							OriginSslProtocols: &awscloudfront.OriginSslProtocols{
								Items:    sslProtocols,
								Quantity: aws.Int64(int64(len(sslProtocols))),
							},
						},
						DomainName: aws.String("origin"),
						Id:         aws.String("origin"),
						OriginPath: aws.String(""),
					},
				},
				Quantity: aws.Int64(2),
			},
			DefaultCacheBehavior: expectedDefaultCacheBehavior,
			CacheBehaviors: &awscloudfront.CacheBehaviors{
				Items: []*awscloudfront.CacheBehavior{
					expectedNewCacheBehavior,
				},
				Quantity: aws.Int64(1),
			},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String("foo"),
	}

	expectedTagResourceInput := &awscloudfront.TagResourceInput{
		Resource: aws.String("arn:aws:cloudfront::1010102030:distribution/ABCABC123456"),
		Tags: &awscloudfront.Tags{
			Items: []*awscloudfront.Tag{
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
			},
		},
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("TagResource", expectedTagResourceInput).Return(noError).Once()

	distribution := Distribution{
		ID:  "mock id",
		ARN: "arn:aws:cloudfront::1010102030:distribution/ABCABC123456",
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
				Behaviors: []Behavior{
					{PathPattern: "/foo", ViewerFnARN: "some-arn", RequestPolicy: "policy"},
				},
			},
		},
		Tags: map[string]string{"foo": "bar"},
	}

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestDelete_Success() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
		expectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
			ETag:         aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
		},
		expectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
			ETag: aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{
				DistributionConfig: disabledDistConfig,
				Status:             aws.String("Deployed"),
			},
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}
	expectedGetDistributionInput := &awscloudfront.GetDistributionInput{
		Id: aws.String("id"),
	}
	expectedDeleteDistributionInput := &awscloudfront.DeleteDistributionInput{
		Id:      aws.String("id"),
		IfMatch: aws.String("etag2"),
	}

	var noError error
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("GetDistribution", expectedGetDistributionInput).Return(noError).Once()
	awsClient.On("DeleteDistribution", expectedDeleteDistributionInput).Return(noError).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToGetDistributionConfig() {
	awsClient := &awsClientMock{}
	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(errors.New("mock err")).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDisableDistribution() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}

	var noError error
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(errors.New("mock err")).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_TimesOutWaitingDistributionDeployment() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
		expectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
			ETag:         aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
		},
		expectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
			ETag: aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{
				DistributionConfig: disabledDistConfig,
				Status:             aws.String("Pending"),
			},
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}
	expectedGetDistributionInput := &awscloudfront.GetDistributionInput{
		Id: aws.String("id"),
	}

	var noError error
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("GetDistribution", expectedGetDistributionInput).Return(errors.New("mock err"))

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Microsecond)
	s.ErrorIs(repo.Delete(Distribution{ID: "id"}), wait.ErrWaitTimeout)
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDeleteDistribution() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
		expectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
			ETag:         aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
		},
		expectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
			ETag: aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{
				DistributionConfig: disabledDistConfig,
				Status:             aws.String("Deployed"),
			},
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}
	expectedGetDistributionInput := &awscloudfront.GetDistributionInput{
		Id: aws.String("id"),
	}
	expectedDeleteDistributionInput := &awscloudfront.DeleteDistributionInput{
		Id:      aws.String("id"),
		IfMatch: aws.String("etag2"),
	}

	var noError error
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("GetDistribution", expectedGetDistributionInput).Return(noError).Once()
	awsClient.On("DeleteDistribution", expectedDeleteDistributionInput).Return(errors.New("mock err")).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionGettingConfig() {
	awsClient := &awsClientMock{}

	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	awsClient.On("GetDistributionConfig", mock.Anything).Return(awsErr).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionDisablingDist() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
		expectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
			ETag:         aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}

	var noError error
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(awsErr).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionWaitingForItToBeDeployed() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
		expectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
			ETag:         aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
		},
		expectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
			ETag: aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{
				DistributionConfig: disabledDistConfig,
				Status:             aws.String("Deployed"),
			},
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}
	expectedGetDistributionInput := &awscloudfront.GetDistributionInput{
		Id: aws.String("id"),
	}

	var noError error
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("GetDistribution", expectedGetDistributionInput).Return(awsErr).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionDeletingIt() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	awsClient := &awsClientMock{
		expectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
			ETag:               aws.String("etag1"),
			DistributionConfig: enabledDistConfig,
		},
		expectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
			ETag:         aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
		},
		expectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
			ETag: aws.String("etag2"),
			Distribution: &awscloudfront.Distribution{
				DistributionConfig: disabledDistConfig,
				Status:             aws.String("Deployed"),
			},
		},
	}

	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: disabledDistConfig,
		Id:                 aws.String("id"),
		IfMatch:            aws.String("etag1"),
	}
	expectedGetDistributionInput := &awscloudfront.GetDistributionInput{
		Id: aws.String("id"),
	}
	expectedDeleteDistributionInput := &awscloudfront.DeleteDistributionInput{
		Id:      aws.String("id"),
		IfMatch: aws.String("etag2"),
	}

	var noError error
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()
	awsClient.On("GetDistribution", expectedGetDistributionInput).Return(noError).Once()
	awsClient.On("DeleteDistribution", expectedDeleteDistributionInput).Return(awsErr).Once()

	repo := NewDistributionRepository(awsClient, testCallerRefFn, time.Minute)
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) Test_baseCacheBehavior_PolicySet() {
	cb := baseCacheBehavior("path", "host", "policy")
	s.Equal("policy", *cb.OriginRequestPolicyId)
}

func (s *DistributionRepositoryTestSuite) Test_baseCacheBehavior_PolicySetToNone() {
	cb := baseCacheBehavior("path", "host", "None")
	s.Nil(cb.OriginRequestPolicyId)
}
