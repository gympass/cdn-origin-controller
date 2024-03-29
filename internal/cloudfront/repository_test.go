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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/test"
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

var noOpPostCreationFunc PostCreationOperationsFunc = func(distribution Distribution) (Distribution, error) {
	return distribution, nil
}

var _ OACRepository = &mockOACRepo{}

type mockOACRepo struct {
	mock.Mock
	expectedSyncOutput   OAC
	expectedDeleteOutput OAC
}

func (m *mockOACRepo) Sync(desired OAC) (OAC, error) {
	args := m.Called(desired)
	return m.expectedSyncOutput, args.Error(0)
}

func (m *mockOACRepo) Delete(toBeDeleted OAC) (OAC, error) {
	args := m.Called(toBeDeleted)
	return m.expectedDeleteOutput, args.Error(0)
}

func TestRunDistributionRepositoryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DistributionRepositoryTestSuite{})
}

type DistributionRepositoryTestSuite struct {
	suite.Suite
	taggingClient *test.MockResourceTaggingAPI
	cfClient      *test.MockCloudFrontAPI
	oacRepo       *mockOACRepo
	cfg           config.Config
}

func (s *DistributionRepositoryTestSuite) SetupTest() {
	s.taggingClient = &test.MockResourceTaggingAPI{}
	s.cfClient = &test.MockCloudFrontAPI{}
	s.oacRepo = &mockOACRepo{}
	s.cfg = config.Config{
		DefaultOriginDomain:  "default.origin",
		CloudFrontPriceClass: awscloudfront.PriceClassPriceClass100,
		CloudFrontWAFARN:     "default-web-acl",
	}
}

func (s *DistributionRepositoryTestSuite) TestARNByGroup_CloudFrontExists() {
	s.taggingClient.ExpectedGetResourcesOutput = &resourcegroupstaggingapi.GetResourcesOutput{
		ResourceTagMappingList: []*resourcegroupstaggingapi.ResourceTagMapping{
			{
				ResourceARN: aws.String("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA"),
			},
		},
	}

	var noError error
	s.taggingClient.On("GetResources", mock.Anything).Return(noError)

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}

	arn, err := repo.ARNByGroup("group")
	s.NoError(err)
	s.Equal("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA", arn)
}

func (s *DistributionRepositoryTestSuite) TestARNByGroup_ErrorGettingResources() {
	s.taggingClient.On("GetResources", mock.Anything).Return(errors.New("mock err"))

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}

	id, err := repo.ARNByGroup("group")
	s.Error(err)
	s.Equal("", id)
}

func (s *DistributionRepositoryTestSuite) TestARNByGroup_DistributionDoesNotExist() {
	s.taggingClient.ExpectedGetResourcesOutput = &resourcegroupstaggingapi.GetResourcesOutput{}

	var noError error
	s.taggingClient.On("GetResources", mock.Anything).Return(noError)

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}

	arn, err := repo.ARNByGroup("group")
	s.ErrorIs(err, ErrDistNotFound)
	s.Equal("", arn)
}

func (s *DistributionRepositoryTestSuite) TestARNByGroup_MoreThanOneCloudFrontExists() {
	s.taggingClient.ExpectedGetResourcesOutput = &resourcegroupstaggingapi.GetResourcesOutput{
		ResourceTagMappingList: []*resourcegroupstaggingapi.ResourceTagMapping{
			{
				ResourceARN: aws.String("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA"),
			},
			{
				ResourceARN: aws.String("arn:aws:cloudfront::000000000000:distribution/BBBBBBBBBBBBBB"),
			},
		},
	}

	var noError error
	s.taggingClient.On("GetResources", mock.Anything).Return(noError)

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}

	arn, err := repo.ARNByGroup("group")
	s.Error(err)
	s.Equal("", arn)
}

func (s *DistributionRepositoryTestSuite) TestCreate_Success() {
	s.cfClient.ExpectedCreateDistributionWithTagsOutput = &awscloudfront.CreateDistributionWithTagsOutput{
		Distribution: &awscloudfront.Distribution{
			Id:         aws.String("L2FB5NP10VU7KL"),
			ARN:        aws.String("arn:aws:cloudfront::123456789012:distribution/L2FB5NP10VU7KL"),
			DomainName: aws.String("aoiweoiwe39d.cloudfront.net"),
		},
	}

	var noError error
	s.cfClient.On("CreateDistributionWithTags", mock.Anything).Return(noError).Once()

	distribution, err := NewDistributionBuilder(
		"test group",
		s.cfg,
	).
		WithOrigin(Origin{Host: "origin", ResponseTimeout: 30}).
		WithAlternateDomains([]string{"test.alias.1", "test.alias.2"}).
		WithWebACL("test web acl").
		AppendTags(map[string]string{"foo": "bar"}).
		WithLogging("test s3", "test prefix").
		WithTLS("test:cert:arn", "test security policy").
		WithIPv6().
		Build()
	s.NoError(err)

	repo := DistRepository{
		CloudFrontClient:          s.cfClient,
		OACRepo:                   s.oacRepo,
		TaggingClient:             s.taggingClient,
		CallerRef:                 testCallerRefFn,
		WaitTimeout:               time.Second,
		RunPostCreationOperations: noOpPostCreationFunc,
		Cfg:                       s.cfg,
	}

	dist, err := repo.Create(distribution)
	s.Equal("L2FB5NP10VU7KL", dist.ID)
	s.Equal("arn:aws:cloudfront::123456789012:distribution/L2FB5NP10VU7KL", dist.ARN)
	s.Equal("aoiweoiwe39d.cloudfront.net", dist.Address)
	s.NoError(err)
}

func (s *DistributionRepositoryTestSuite) TestCreate_ErrorWhenCreatingDistribution() {
	s.cfClient.On("CreateDistributionWithTags", mock.Anything).Return(errors.New("mock err")).Once()

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

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	dist, err := repo.Create(distribution)
	s.Equal(Distribution{}, dist)
	s.Error(err)
}

func (s *DistributionRepositoryTestSuite) TestSync_CantFetchDistribution() {
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(errors.New("mock err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	gotDist, err := repo.Sync(Distribution{})
	s.Error(err)
	s.Equal(Distribution{}, gotDist)
}

func (s *DistributionRepositoryTestSuite) TestSync_CantUpdateDistribution() {
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
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
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(errors.New("mock err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	gotDist, err := repo.Sync(Distribution{})
	s.Error(err)
	s.Equal(Distribution{}, gotDist)
}

func (s *DistributionRepositoryTestSuite) TestSync_CantSaveTags() {
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
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
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("TagResource", mock.Anything).Return(errors.New("mock err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	gotDist, err := repo.Sync(Distribution{})
	s.Error(err)
	s.Equal(Distribution{}, gotDist)
}

func (s *DistributionRepositoryTestSuite) TestSync_OriginDoesNotExistYet() {
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
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

	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		Distribution: &awscloudfront.Distribution{
			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("TagResource", mock.Anything).Return(noError).Once()

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

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	_, err := repo.Sync(distribution)
	s.NoError(err)
}

func (s *DistributionRepositoryTestSuite) TestSync_OriginAlreadyExists() {
	someIncorrectOrigin := &awscloudfront.Origin{Id: aws.String("origin"), DomainName: aws.String("incorrect domain name")}

	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
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

	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		Distribution: &awscloudfront.Distribution{
			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("TagResource", mock.Anything).Return(noError).Once()

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

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	_, err := repo.Sync(distribution)
	s.NoError(err)
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
		CachePolicyId:              aws.String("cache-policy"),
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
		CachePolicyId:              aws.String("cache-policy"),
		Compress:                   aws.Bool(true),
		FieldLevelEncryptionId:     aws.String(""),
		LambdaFunctionAssociations: &awscloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
		OriginRequestPolicyId:      aws.String("policy"),
		PathPattern:                aws.String("/very/high/precedence/path/very/lengthy/indeed"),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String("origin"),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins: &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{
				Quantity: aws.Int64(2),
				Items: []*awscloudfront.CacheBehavior{
					higherPrecedenceExistingBehavior,
					lowerPrecedenceExistingBehavior,
				},
			},
			CallerReference:      aws.String(testCallerRefFn()),
			DefaultRootObject:    aws.String("/"),
			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
			Restrictions:         &awscloudfront.Restrictions{},
		},
	}

	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		Distribution: &awscloudfront.Distribution{
			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
		},
	}

	var noError error

	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("TagResource", mock.Anything).Return(noError).Once()

	distribution := Distribution{
		ID: "mock id",
		CustomOrigins: []Origin{
			{
				Host:            "origin",
				ResponseTimeout: 30,
				Behaviors: []Behavior{
					{PathPattern: "/mid-sized/path/with/medium/precedence", RequestPolicy: "policy", CachePolicy: "cache-policy"},
					{PathPattern: "/low/precedence/path", RequestPolicy: "policy", CachePolicy: "cache-policy"},
					{PathPattern: "/very/high/precedence/path/very/lengthy/indeed", RequestPolicy: "policy", CachePolicy: "cache-policy"},
				},
			},
		},
		DefaultOrigin: Origin{
			Host:            "default.origin",
			ResponseTimeout: 30,
		},
	}

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	_, err := repo.Sync(distribution)
	s.NoError(err)
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
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
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

	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		Distribution: &awscloudfront.Distribution{
			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("TagResource", mock.Anything).Return(noError).Once()

	s.oacRepo.On("Sync", mock.Anything).Return(noError)

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}

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
				Behaviors:       []Behavior{{PathPattern: "/*", RequestPolicy: "policy", CachePolicy: "cache-policy"}},
			},
		},
	}
	_, err := repo.Sync(distribution)
	s.NoError(err)
}

func (s *DistributionRepositoryTestSuite) TestUpdate_ShouldSyncOneOACAndDeleteOneOAC() {
	origins := &awscloudfront.Origins{
		Items: []*awscloudfront.Origin{
			{OriginAccessControlId: aws.String("some oac"), Id: aws.String("host")},
			{OriginAccessControlId: aws.String("another oac"), Id: aws.String(" some other host")},
		},
	}
	distConfig := &awscloudfront.DistributionConfig{
		Origins:              origins,
		CallerReference:      aws.String(testCallerRefFn()),
		DefaultRootObject:    aws.String("/"),
		CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
		Restrictions:         &awscloudfront.Restrictions{},
	}

	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		DistributionConfig: distConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		Distribution: &awscloudfront.Distribution{
			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
			DistributionConfig: distConfig,
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("TagResource", mock.Anything).Return(noError).Once()

	s.oacRepo.On("Delete", mock.Anything).Return(noError).Once()
	s.oacRepo.On("Sync", mock.Anything).Return(noError).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	_, err := repo.Sync(Distribution{
		ID: "id",
		CustomOrigins: []Origin{{
			Host:   "host",
			Access: OriginAccessBucket,
			OAC:    OAC{ID: "some oac"},
		}},
	})
	s.NoError(err)
}

func (s *DistributionRepositoryTestSuite) TestDelete_SuccessWithPublicOrigins() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}

	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Deployed"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("DeleteDistribution", mock.Anything).Return(noError).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_SuccessWithS3Origins() {
	origins := &awscloudfront.Origins{
		Items: []*awscloudfront.Origin{
			{OriginAccessControlId: aws.String("some oac")},
			{OriginAccessControlId: aws.String("another oac")},
		},
	}
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true), Origins: origins}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false), Origins: origins}

	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}

	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Deployed"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("DeleteDistribution", mock.Anything).Return(noError).Once()

	s.oacRepo.On("Delete", mock.Anything).Return(noError).Twice()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDeleteOACs() {
	origins := &awscloudfront.Origins{
		Items: []*awscloudfront.Origin{
			{OriginAccessControlId: aws.String("some oac")},
			{OriginAccessControlId: aws.String("another oac")},
		},
	}
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true), Origins: origins}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false), Origins: origins}

	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}

	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Deployed"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("DeleteDistribution", mock.Anything).Return(noError).Once()

	s.oacRepo.On("Delete", mock.Anything).Return(errors.New("some err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToGetDistributionConfig() {
	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
		Id: aws.String("id"),
	}
	s.cfClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(errors.New("mock err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDisableDistribution() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
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
	s.cfClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(errors.New("mock err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_TimesOutWaitingDistributionDeployment() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}
	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Pending"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(errors.New("mock err"))

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.ErrorIs(repo.Delete(Distribution{ID: "id"}), context.DeadlineExceeded)
}

func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDeleteDistribution() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}
	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Deployed"),
		},
	}

	var noError error
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("DeleteDistribution", mock.Anything).Return(errors.New("mock err")).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.Error(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionGettingConfig() {
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(awsErr).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionDisablingDist() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}

	var noError error
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(awsErr).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionWaitingForItToBeDeployed() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}
	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Deployed"),
		},
	}

	var noError error
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(awsErr).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionDeletingIt() {
	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
	s.cfClient.ExpectedGetDistributionConfigOutput = &awscloudfront.GetDistributionConfigOutput{
		ETag:               aws.String("etag1"),
		DistributionConfig: enabledDistConfig,
	}
	s.cfClient.ExpectedUpdateDistributionOutput = &awscloudfront.UpdateDistributionOutput{
		ETag:         aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
	}
	s.cfClient.ExpectedGetDistributionOutput = &awscloudfront.GetDistributionOutput{
		ETag: aws.String("etag2"),
		Distribution: &awscloudfront.Distribution{
			DistributionConfig: disabledDistConfig,
			Status:             aws.String("Deployed"),
		},
	}

	var noError error
	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
	s.cfClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	s.cfClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("GetDistribution", mock.Anything).Return(noError).Once()
	s.cfClient.On("DeleteDistribution", mock.Anything).Return(awsErr).Once()

	repo := DistRepository{
		CloudFrontClient: s.cfClient,
		OACRepo:          s.oacRepo,
		TaggingClient:    s.taggingClient,
		CallerRef:        testCallerRefFn,
		WaitTimeout:      time.Second,
		Cfg:              s.cfg,
	}
	s.NoError(repo.Delete(Distribution{ID: "id"}))
}

func (s *DistributionRepositoryTestSuite) Test_baseCacheBehavior_PolicySet() {
	cb := baseCacheBehavior(
		Behavior{
			OriginHost:    "host",
			PathPattern:   "path",
			RequestPolicy: "b2884449-e4de-46a7-ac36-70bc7f1ddd6d",
		},
	)
	s.Equal("b2884449-e4de-46a7-ac36-70bc7f1ddd6d", *cb.OriginRequestPolicyId)
}

func (s *DistributionRepositoryTestSuite) Test_baseCacheBehavior_PolicySetToNone() {
	cb := baseCacheBehavior(
		Behavior{
			OriginHost:    "host",
			PathPattern:   "path",
			RequestPolicy: "None",
		},
	)
	s.Nil(cb.OriginRequestPolicyId)
}
