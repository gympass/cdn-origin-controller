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

package cloudfront_test

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/cloudfront"
)

const (
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-cache-policies.html
	cachingDisabledPolicyID = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"
	// https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/using-managed-origin-request-policies.html
	allViewerOriginRequestPolicyID = "216adef6-5c7f-47e4-b989-5492eafa07d3"
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

	testCallerRefFn = func() string { return "test caller ref" }
)

type awsClientMock struct {
	mock.Mock
	cloudfrontiface.CloudFrontAPI
	expectedGetDistributionConfigOutput *awscloudfront.GetDistributionConfigOutput
	expectedUpdateDistributionOutput    *awscloudfront.UpdateDistributionOutput
}

func (c *awsClientMock) GetDistributionConfig(in *awscloudfront.GetDistributionConfigInput) (*awscloudfront.GetDistributionConfigOutput, error) {
	args := c.Called(in)
	return c.expectedGetDistributionConfigOutput, args.Error(0)
}

func (c *awsClientMock) UpdateDistribution(in *awscloudfront.UpdateDistributionInput) (*awscloudfront.UpdateDistributionOutput, error) {
	args := c.Called(in)
	return c.expectedUpdateDistributionOutput, args.Error(0)
}

func (c *awsClientMock) CreateDistributionWithTags(in *awscloudfront.CreateDistributionWithTagsInput) (*awscloudfront.CreateDistributionWithTagsOutput, error) {
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

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Create_Success() {
	awsClient := &awsClientMock{}

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
					Items:    []*awscloudfront.CacheBehavior{},
					Quantity: aws.Int64(0),
				},
				Comment: aws.String("test description"),
				DefaultCacheBehavior: &awscloudfront.DefaultCacheBehavior{
					CachePolicyId:         aws.String(cachingDisabledPolicyID),
					OriginRequestPolicyId: aws.String(allViewerOriginRequestPolicyID),
					TargetOriginId:        aws.String("default.origin"),
					ViewerProtocolPolicy:  aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
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

	distribution := cloudfront.NewDistributionBuilder(
		cloudfront.Origin{Host: "origin", ResponseTimeout: 30},
		"default.origin",
		"test description",
		awscloudfront.PriceClassPriceClass100,
		"test group",
	).
		WithAlternateDomains([]string{"test.alias.1", "test.alias.2"}).
		WithWebACL("test web acl").
		WithTags(map[string]string{"foo": "bar"}).
		WithLogging("test s3", "test prefix").
		WithTLS("test:cert:arn", "test security policy").
		WithIPv6().
		Build()

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.NoError(repo.Create(distribution))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Create_ErrorWhenCreatingDistribution() {
	awsClient := &awsClientMock{}
	awsClient.On("CreateDistributionWithTags", mock.Anything).Return(errors.New("mock err")).Once()

	distribution := cloudfront.Distribution{
		ID: "mock id",
		DefaultOrigin: cloudfront.Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigin: cloudfront.Origin{
			Host:            "origin",
			ResponseTimeout: 30,
		},
	}

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.Error(repo.Create(distribution))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_CantFetchDistribution() {
	awsClient := &awsClientMock{}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(errors.New("mock err")).Once()

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.Error(repo.Sync(cloudfront.Distribution{}))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_CantUpdateDistribution() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", mock.Anything).Return(errors.New("mock err")).Once()

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.Error(repo.Sync(cloudfront.Distribution{}))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_OriginDoesNotExistYet() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Comment:       aws.String(""),
			HttpVersion:   aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled: aws.Bool(false),
			WebACLId:      aws.String(""),
			PriceClass:    aws.String(""),
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
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()

	distribution := cloudfront.Distribution{
		ID: "mock id",
		DefaultOrigin: cloudfront.Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigin: cloudfront.Origin{
			Host:            "origin",
			ResponseTimeout: 30,
		},
	}

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_OriginAlreadyExists() {
	someIncorrectOrigin := &awscloudfront.Origin{Id: aws.String("origin"), DomainName: aws.String("incorrect domain name")}

	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Items: []*awscloudfront.Origin{someIncorrectOrigin}, Quantity: aws.Int64(1)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Comment:       aws.String(""),
			HttpVersion:   aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled: aws.Bool(false),
			WebACLId:      aws.String(""),
			PriceClass:    aws.String(""),
			Origins: &awscloudfront.Origins{
				Items: []*awscloudfront.Origin{
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
					defaultOrigin,
				},
				Quantity: aws.Int64(2),
			},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()

	distribution := cloudfront.Distribution{
		ID: "mock id",
		DefaultOrigin: cloudfront.Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigin: cloudfront.Origin{
			Host:            "origin",
			ResponseTimeout: 30,
		},
	}

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_BehaviorDoesNotExistYet() {
	lowerPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{PathPattern: aws.String("/low/precedence/path")}
	higherPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{PathPattern: aws.String("/very/high/precedence/path/very/lengthy/indeed")}
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(2), Items: []*awscloudfront.CacheBehavior{higherPrecedenceExistingBehavior, lowerPrecedenceExistingBehavior}},
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
		OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
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
			Comment:       aws.String(""),
			HttpVersion:   aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled: aws.Bool(false),
			WebACLId:      aws.String(""),
			PriceClass:    aws.String(""),
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

	distribution := cloudfront.Distribution{
		ID: "mock id",
		CustomOrigin: cloudfront.Origin{
			Host:            "origin",
			ResponseTimeout: 30,
			Behaviors:       []cloudfront.Behavior{{PathPattern: "/mid-sized/path/with/medium/precedence"}},
		},
		DefaultOrigin: cloudfront.Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
	}

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.NoError(repo.Sync(distribution))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_BehaviorAlreadyExists() {
	existingOrigins := &awscloudfront.Origins{
		Items: []*awscloudfront.Origin{
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
		Quantity: aws.Int64(1),
	}

	someIncorrectBehavior := &awscloudfront.CacheBehavior{PathPattern: aws.String("/*"), SmoothStreaming: aws.Bool(true)}
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        existingOrigins,
			CacheBehaviors: &awscloudfront.CacheBehaviors{Items: []*awscloudfront.CacheBehavior{someIncorrectBehavior}, Quantity: aws.Int64(1)},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
			Aliases: &awscloudfront.Aliases{
				Items:    []*string{},
				Quantity: aws.Int64(0),
			},
			Comment:       aws.String(""),
			HttpVersion:   aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled: aws.Bool(false),
			WebACLId:      aws.String(""),
			PriceClass:    aws.String(""),
			Origins:       existingOrigins,
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
						OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
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

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.NoError(repo.Sync(cloudfront.Distribution{ID: "mock id", CustomOrigin: cloudfront.Origin{Host: "origin", ResponseTimeout: 30, Behaviors: []cloudfront.Behavior{{PathPattern: "/*"}}}}))
}

func (s *DistributionRepositoryTestSuite) TestDistributionRepository_Sync_WithViewerFunction() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
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
		OriginRequestPolicyId:      aws.String(allViewerOriginRequestPolicyID),
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
			Comment:       aws.String(""),
			HttpVersion:   aws.String(awscloudfront.HttpVersionHttp2),
			IsIPV6Enabled: aws.Bool(false),
			WebACLId:      aws.String(""),
			PriceClass:    aws.String(""),
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
			CacheBehaviors: &awscloudfront.CacheBehaviors{
				Items: []*awscloudfront.CacheBehavior{
					expectedNewCacheBehavior,
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

	distribution := cloudfront.Distribution{
		ID: "mock id",
		DefaultOrigin: cloudfront.Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
		CustomOrigin: cloudfront.Origin{
			Host:            "origin",
			ResponseTimeout: 30,
			Behaviors: []cloudfront.Behavior{
				{PathPattern: "/foo", ViewerFnARN: "some-arn"},
			},
		},
	}

	repo := cloudfront.NewDistributionRepository(awsClient, testCallerRefFn)
	s.NoError(repo.Sync(distribution))
}
