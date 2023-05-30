// // Copyright (c) 2021 GPBR Participacoes LTDA.
// //
// // Permission is hereby granted, free of charge, to any person obtaining a copy of
// // this software and associated documentation files (the "Software"), to deal in
// // the Software without restriction, including without limitation the rights to
// // use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// // the Software, and to permit persons to whom the Software is furnished to do so,
// // subject to the following conditions:
// //
// // The above copyright notice and this permission notice shall be included in all
// // copies or substantial portions of the Software.
// //
// // THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// // IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// // FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// // COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// // IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// // CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
package cloudfront

//
//import (
//	"context"
//	"errors"
//	"testing"
//	"time"
//
//	"github.com/aws/aws-sdk-go/aws"
//	"github.com/aws/aws-sdk-go/aws/awserr"
//	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
//	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
//	"github.com/stretchr/testify/mock"
//	"github.com/stretchr/testify/suite"
//
//	"github.com/Gympass/cdn-origin-controller/internal/test"
//)
//
//var (
//	sslProtocols = []*string{
//		aws.String("SSLv3"),
//		aws.String("TLSv1"),
//		aws.String("TLSv1.1"),
//		aws.String("TLSv1.2"),
//	}
//	defaultOrigin = &awscloudfront.Origin{
//		CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
//		CustomOriginConfig: &awscloudfront.CustomOriginConfig{
//			HTTPPort:               aws.Int64(80),
//			HTTPSPort:              aws.Int64(443),
//			OriginKeepaliveTimeout: aws.Int64(5),
//			OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
//			OriginReadTimeout:      aws.Int64(30),
//			OriginSslProtocols: &awscloudfront.OriginSslProtocols{
//				Items:    sslProtocols,
//				Quantity: aws.Int64(int64(len(sslProtocols))),
//			},
//		},
//		DomainName: aws.String("default.origin"),
//		Id:         aws.String("default.origin"),
//		OriginPath: aws.String(""),
//	}
//	testCallerRefFn = func() string { return "test caller ref" }
//)
//
//func TestRunDistributionRepositoryTestSuite(t *testing.T) {
//	t.Parallel()
//	suite.Run(t, &DistributionRepositoryTestSuite{})
//}
//
//type DistributionRepositoryTestSuite struct {
//	suite.Suite
//}
//
//func (s *DistributionRepositoryTestSuite) TestARNByGroup_CloudFrontExists() {
//	taggingClient := &test.MockResourceTaggingAPI{
//		ExpectedGetResourcesOutput: &resourcegroupstaggingapi.GetResourcesOutput{
//			ResourceTagMappingList: []*resourcegroupstaggingapi.ResourceTagMapping{
//				{
//					ResourceARN: aws.String("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA"),
//				},
//			},
//		},
//	}
//
//	var noError error
//	taggingClient.On("GetResources", mock.Anything).Return(noError)
//
//	repo := NewDistributionRepository(&test.MockCloudFrontAPI{}, taggingClient, testCallerRefFn, time.Second)
//
//	arn, err := repo.ARNByGroup("group")
//	s.NoError(err)
//	s.Equal("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA", arn)
//}
//
//func (s *DistributionRepositoryTestSuite) TestARNByGroup_ErrorGettingResources() {
//	taggingClient := &test.MockResourceTaggingAPI{}
//
//	taggingClient.On("GetResources", mock.Anything).Return(errors.New("mock err"))
//
//	repo := NewDistributionRepository(&test.MockCloudFrontAPI{}, taggingClient, testCallerRefFn, time.Second)
//
//	id, err := repo.ARNByGroup("group")
//	s.Error(err)
//	s.Equal("", id)
//}
//
//func (s *DistributionRepositoryTestSuite) TestARNByGroup_DistributionDoesNotExist() {
//	taggingClient := &test.MockResourceTaggingAPI{
//		ExpectedGetResourcesOutput: &resourcegroupstaggingapi.GetResourcesOutput{},
//	}
//
//	var noError error
//	taggingClient.On("GetResources", mock.Anything).Return(noError)
//
//	repo := NewDistributionRepository(&test.MockCloudFrontAPI{}, taggingClient, testCallerRefFn, time.Second)
//
//	arn, err := repo.ARNByGroup("group")
//	s.ErrorIs(err, ErrDistNotFound)
//	s.Equal("", arn)
//}
//
//func (s *DistributionRepositoryTestSuite) TestARNByGroup_MoreThanOneCloudFrontExists() {
//	taggingClient := &test.MockResourceTaggingAPI{
//		ExpectedGetResourcesOutput: &resourcegroupstaggingapi.GetResourcesOutput{
//			ResourceTagMappingList: []*resourcegroupstaggingapi.ResourceTagMapping{
//				{
//					ResourceARN: aws.String("arn:aws:cloudfront::000000000000:distribution/AAAAAAAAAAAAAA"),
//				},
//				{
//					ResourceARN: aws.String("arn:aws:cloudfront::000000000000:distribution/BBBBBBBBBBBBBB"),
//				},
//			},
//		},
//	}
//
//	var noError error
//	taggingClient.On("GetResources", mock.Anything).Return(noError)
//
//	repo := NewDistributionRepository(&test.MockCloudFrontAPI{}, taggingClient, testCallerRefFn, time.Second)
//
//	arn, err := repo.ARNByGroup("group")
//	s.Error(err)
//	s.Equal("", arn)
//}
//
//func (s *DistributionRepositoryTestSuite) TestCreate_Success() {
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedCreateDistributionWithTagsOutput: &awscloudfront.CreateDistributionWithTagsOutput{
//			Distribution: &awscloudfront.Distribution{
//				Id:         aws.String("L2FB5NP10VU7KL"),
//				ARN:        aws.String("arn:aws:cloudfront::123456789012:distribution/L2FB5NP10VU7KL"),
//				DomainName: aws.String("aoiweoiwe39d.cloudfront.net"),
//			},
//		},
//	}
//
//	var noError error
//	awsClient.On("CreateDistributionWithTags", mock.Anything).Return(noError).Once()
//
//	distribution, err := NewDistributionBuilder(
//		"default.origin",
//		"test description",
//		awscloudfront.PriceClassPriceClass100,
//		"test group",
//		"default-web-acl",
//	).
//		WithOrigin(Origin{Host: "origin", ResponseTimeout: 30}).
//		WithAlternateDomains([]string{"test.alias.1", "test.alias.2"}).
//		WithWebACL("test web acl").
//		AppendTags(map[string]string{"foo": "bar"}).
//		WithLogging("test s3", "test prefix").
//		WithTLS("test:cert:arn", "test security policy").
//		WithIPv6().
//		Build()
//	s.NoError(err)
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	dist, err := repo.Create(distribution)
//	s.Equal(dist.ID, "L2FB5NP10VU7KL")
//	s.Equal(dist.ARN, "arn:aws:cloudfront::123456789012:distribution/L2FB5NP10VU7KL")
//	s.Equal(dist.Address, "aoiweoiwe39d.cloudfront.net")
//	s.NoError(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestCreate_ErrorWhenCreatingDistribution() {
//	awsClient := &test.MockCloudFrontAPI{}
//	awsClient.On("CreateDistributionWithTags", mock.Anything).Return(errors.New("mock err")).Once()
//
//	distribution := Distribution{
//		ID: "mock id",
//		DefaultOrigin: Origin{
//			Host:            "default.origin",
//			ResponseTimeout: 30,
//		},
//		CustomOrigins: []Origin{
//			{
//				Host:            "origin",
//				ResponseTimeout: 30,
//			},
//		},
//	}
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	dist, err := repo.Create(distribution)
//	s.Equal(Distribution{}, dist)
//	s.Error(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_CantFetchDistribution() {
//	awsClient := &test.MockCloudFrontAPI{}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(errors.New("mock err")).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	gotDist, err := repo.Sync(Distribution{})
//	s.Error(err)
//	s.Equal(Distribution{}, gotDist)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_CantUpdateDistribution() {
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String(""),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
//			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
//			Enabled:              aws.Bool(true),
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(errors.New("mock err")).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	gotDist, err := repo.Sync(Distribution{})
//	s.Error(err)
//	s.Equal(Distribution{}, gotDist)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_CantSaveTags() {
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String(""),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
//			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
//			Enabled:              aws.Bool(true),
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("TagResource", mock.Anything).Return(errors.New("mock err")).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	gotDist, err := repo.Sync(Distribution{})
//	s.Error(err)
//	s.Equal(Distribution{}, gotDist)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_OriginDoesNotExistYet() {
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String(""),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
//			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
//			Enabled:              aws.Bool(true),
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	expectedUpdateDistributionOut := &awscloudfront.UpdateDistributionOutput{
//		Distribution: &awscloudfront.Distribution{
//			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput,
//		ExpectedUpdateDistributionOutput:    expectedUpdateDistributionOut,
//	}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("TagResource", mock.Anything).Return(noError).Once()
//
//	distribution := Distribution{
//		ID: "mock id",
//		DefaultOrigin: Origin{
//			Host:            "default.origin",
//			ResponseTimeout: 30,
//		},
//		CustomOrigins: []Origin{
//			{
//				Host:            "origin",
//				ResponseTimeout: 30,
//			},
//		},
//	}
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	_, err := repo.Sync(distribution)
//	s.NoError(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_OriginAlreadyExists() {
//	someIncorrectOrigin := &awscloudfront.Origin{Id: aws.String("origin"), DomainName: aws.String("incorrect domain name")}
//
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String(""),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins:              &awscloudfront.Origins{Items: []*awscloudfront.Origin{someIncorrectOrigin}, Quantity: aws.Int64(1)},
//			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	expectedUpdateDistributionOut := &awscloudfront.UpdateDistributionOutput{
//		Distribution: &awscloudfront.Distribution{
//			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput,
//		ExpectedUpdateDistributionOutput:    expectedUpdateDistributionOut,
//	}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("TagResource", mock.Anything).Return(noError).Once()
//
//	distribution := Distribution{
//		ID: "mock id",
//		DefaultOrigin: Origin{
//			Host:            "default.origin",
//			ResponseTimeout: 30,
//		},
//		CustomOrigins: []Origin{
//			{
//				Host:            "origin",
//				ResponseTimeout: 30,
//			},
//		},
//	}
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	_, err := repo.Sync(distribution)
//	s.NoError(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_BehaviorDoesNotExistYet() {
//
//	lowerPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{
//		AllowedMethods: &awscloudfront.AllowedMethods{
//			Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
//			Quantity: aws.Int64(7),
//			CachedMethods: &awscloudfront.CachedMethods{
//				Items:    aws.StringSlice([]string{"GET", "HEAD"}),
//				Quantity: aws.Int64(2),
//			},
//		},
//		CachePolicyId:              aws.String("cache-policy"),
//		Compress:                   aws.Bool(true),
//		FieldLevelEncryptionId:     aws.String(""),
//		LambdaFunctionAssociations: &awscloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
//		OriginRequestPolicyId:      aws.String("policy"),
//		PathPattern:                aws.String("/low/precedence/path"),
//		SmoothStreaming:            aws.Bool(false),
//		TargetOriginId:             aws.String("origin"),
//		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
//	}
//
//	higherPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{
//		AllowedMethods: &awscloudfront.AllowedMethods{
//			Items:    aws.StringSlice([]string{"GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"}),
//			Quantity: aws.Int64(7),
//			CachedMethods: &awscloudfront.CachedMethods{
//				Items:    aws.StringSlice([]string{"GET", "HEAD"}),
//				Quantity: aws.Int64(2),
//			},
//		},
//		CachePolicyId:              aws.String("cache-policy"),
//		Compress:                   aws.Bool(true),
//		FieldLevelEncryptionId:     aws.String(""),
//		LambdaFunctionAssociations: &awscloudfront.LambdaFunctionAssociations{Quantity: aws.Int64(0)},
//		OriginRequestPolicyId:      aws.String("policy"),
//		PathPattern:                aws.String("/very/high/precedence/path/very/lengthy/indeed"),
//		SmoothStreaming:            aws.Bool(false),
//		TargetOriginId:             aws.String("origin"),
//		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
//	}
//
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String(""),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins: &awscloudfront.Origins{Quantity: aws.Int64(0)},
//			CacheBehaviors: &awscloudfront.CacheBehaviors{
//				Quantity: aws.Int64(2),
//				Items: []*awscloudfront.CacheBehavior{
//					higherPrecedenceExistingBehavior,
//					lowerPrecedenceExistingBehavior,
//				},
//			},
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	expectedUpdateDistributionOut := &awscloudfront.UpdateDistributionOutput{
//		Distribution: &awscloudfront.Distribution{
//			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput,
//		ExpectedUpdateDistributionOutput:    expectedUpdateDistributionOut,
//	}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("TagResource", mock.Anything).Return(noError).Once()
//
//	distribution := Distribution{
//		ID: "mock id",
//		CustomOrigins: []Origin{
//			{
//				Host:            "origin",
//				ResponseTimeout: 30,
//				Behaviors: []Behavior{
//					{PathPattern: "/mid-sized/path/with/medium/precedence", RequestPolicy: "policy", CachePolicy: "cache-policy"},
//					{PathPattern: "/low/precedence/path", RequestPolicy: "policy", CachePolicy: "cache-policy"},
//					{PathPattern: "/very/high/precedence/path/very/lengthy/indeed", RequestPolicy: "policy", CachePolicy: "cache-policy"},
//				},
//			},
//		},
//		DefaultOrigin: Origin{
//			Host:            "default.origin",
//			ResponseTimeout: 30,
//		},
//	}
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	_, err := repo.Sync(distribution)
//	s.NoError(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_BehaviorAlreadyExists() {
//	existingOrigins := &awscloudfront.Origins{
//		Items: []*awscloudfront.Origin{
//			defaultOrigin,
//			{
//				CustomHeaders: &awscloudfront.CustomHeaders{Quantity: aws.Int64(0)},
//				CustomOriginConfig: &awscloudfront.CustomOriginConfig{
//					HTTPPort:               aws.Int64(80),
//					HTTPSPort:              aws.Int64(443),
//					OriginKeepaliveTimeout: aws.Int64(5),
//					OriginProtocolPolicy:   aws.String(awscloudfront.OriginProtocolPolicyMatchViewer),
//					OriginReadTimeout:      aws.Int64(30),
//					OriginSslProtocols: &awscloudfront.OriginSslProtocols{
//						Items:    sslProtocols,
//						Quantity: aws.Int64(int64(len(sslProtocols))),
//					},
//				},
//				DomainName: aws.String("origin"),
//				Id:         aws.String("origin"),
//				OriginPath: aws.String(""),
//			},
//		},
//		Quantity: aws.Int64(2),
//	}
//
//	someIncorrectBehavior := &awscloudfront.CacheBehavior{PathPattern: aws.String("/*"), SmoothStreaming: aws.Bool(true)}
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String(""),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins:              existingOrigins,
//			CacheBehaviors:       &awscloudfront.CacheBehaviors{Items: []*awscloudfront.CacheBehavior{someIncorrectBehavior}, Quantity: aws.Int64(1)},
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	expectedUpdateDistributionOut := &awscloudfront.UpdateDistributionOutput{
//		Distribution: &awscloudfront.Distribution{
//			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput,
//		ExpectedUpdateDistributionOutput:    expectedUpdateDistributionOut,
//	}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("TagResource", mock.Anything).Return(noError).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//
//	distribution := Distribution{
//		ID: "mock id",
//		DefaultOrigin: Origin{
//			Host:            "default.origin",
//			ResponseTimeout: 30,
//		},
//		CustomOrigins: []Origin{
//			{
//				Host:            "origin",
//				ResponseTimeout: 30,
//				Behaviors:       []Behavior{{PathPattern: "/*", RequestPolicy: "policy", CachePolicy: "cache-policy"}},
//			},
//		},
//	}
//	_, err := repo.Sync(distribution)
//	s.NoError(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestSync_WithViewerFunction() {
//	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
//		ETag: aws.String("foo"),
//		DistributionConfig: &awscloudfront.DistributionConfig{
//			Origins:              &awscloudfront.Origins{Quantity: aws.Int64(0)},
//			CacheBehaviors:       &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
//			CallerReference:      aws.String(testCallerRefFn()),
//			DefaultRootObject:    aws.String("/"),
//			CustomErrorResponses: &awscloudfront.CustomErrorResponses{},
//			Restrictions:         &awscloudfront.Restrictions{},
//		},
//	}
//
//	expectedUpdateDistributionOut := &awscloudfront.UpdateDistributionOutput{
//		Distribution: &awscloudfront.Distribution{
//			Id: aws.String("id"), ARN: aws.String("arn"), DomainName: aws.String("domain"),
//		},
//	}
//
//	var noError error
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: expectedDistributionConfigOutput,
//		ExpectedUpdateDistributionOutput:    expectedUpdateDistributionOut,
//	}
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("TagResource", mock.Anything).Return(noError).Once()
//
//	distribution := Distribution{
//		ID:  "mock id",
//		ARN: "arn:aws:cloudfront::1010102030:distribution/ABCABC123456",
//		DefaultOrigin: Origin{
//			Host:            "default.origin",
//			ResponseTimeout: 30,
//		},
//		CustomOrigins: []Origin{
//			{
//				Host:            "origin",
//				ResponseTimeout: 30,
//				Behaviors: []Behavior{
//					{PathPattern: "/foo", ViewerFnARN: "some-arn", RequestPolicy: "policy", CachePolicy: "cache-policy"},
//				},
//			},
//		},
//		Tags: map[string]string{"foo": "bar"},
//	}
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	_, err := repo.Sync(distribution)
//	s.NoError(err)
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_Success() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//		ExpectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
//			ETag:         aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
//		},
//		ExpectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
//			ETag: aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{
//				DistributionConfig: disabledDistConfig,
//				Status:             aws.String("Deployed"),
//			},
//		},
//	}
//
//	var noError error
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("GetDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("DeleteDistribution", mock.Anything).Return(noError).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.NoError(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_FailsToGetDistributionConfig() {
//	awsClient := &test.MockCloudFrontAPI{}
//	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
//		Id: aws.String("id"),
//	}
//	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(errors.New("mock err")).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.Error(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDisableDistribution() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//	}
//
//	expectedGetDistributionConfigInput := &awscloudfront.GetDistributionConfigInput{
//		Id: aws.String("id"),
//	}
//	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
//		DistributionConfig: disabledDistConfig,
//		Id:                 aws.String("id"),
//		IfMatch:            aws.String("etag1"),
//	}
//
//	var noError error
//	awsClient.On("GetDistributionConfig", expectedGetDistributionConfigInput).Return(noError).Once()
//	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(errors.New("mock err")).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.Error(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_TimesOutWaitingDistributionDeployment() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//		ExpectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
//			ETag:         aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
//		},
//		ExpectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
//			ETag: aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{
//				DistributionConfig: disabledDistConfig,
//				Status:             aws.String("Pending"),
//			},
//		},
//	}
//
//	var noError error
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("GetDistribution", mock.Anything).Return(errors.New("mock err"))
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Microsecond)
//	s.ErrorIs(repo.Delete(Distribution{ID: "id"}), context.DeadlineExceeded)
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_FailsToDeleteDistribution() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//		ExpectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
//			ETag:         aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
//		},
//		ExpectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
//			ETag: aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{
//				DistributionConfig: disabledDistConfig,
//				Status:             aws.String("Deployed"),
//			},
//		},
//	}
//
//	var noError error
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("GetDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("DeleteDistribution", mock.Anything).Return(errors.New("mock err")).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.Error(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionGettingConfig() {
//	awsClient := &test.MockCloudFrontAPI{}
//
//	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(awsErr).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.NoError(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionDisablingDist() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//		ExpectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
//			ETag:         aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
//		},
//	}
//
//	var noError error
//	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(awsErr).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.NoError(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionWaitingForItToBeDeployed() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//		ExpectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
//			ETag:         aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
//		},
//		ExpectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
//			ETag: aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{
//				DistributionConfig: disabledDistConfig,
//				Status:             aws.String("Deployed"),
//			},
//		},
//	}
//
//	var noError error
//	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("GetDistribution", mock.Anything).Return(awsErr).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.NoError(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) TestDelete_NoSuchDistributionDeletingIt() {
//	enabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(true)}
//	disabledDistConfig := &awscloudfront.DistributionConfig{Enabled: aws.Bool(false)}
//	awsClient := &test.MockCloudFrontAPI{
//		ExpectedGetDistributionConfigOutput: &awscloudfront.GetDistributionConfigOutput{
//			ETag:               aws.String("etag1"),
//			DistributionConfig: enabledDistConfig,
//		},
//		ExpectedUpdateDistributionOutput: &awscloudfront.UpdateDistributionOutput{
//			ETag:         aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{DistributionConfig: disabledDistConfig},
//		},
//		ExpectedGetDistributionOutput: &awscloudfront.GetDistributionOutput{
//			ETag: aws.String("etag2"),
//			Distribution: &awscloudfront.Distribution{
//				DistributionConfig: disabledDistConfig,
//				Status:             aws.String("Deployed"),
//			},
//		},
//	}
//
//	var noError error
//	awsErr := awserr.New(awscloudfront.ErrCodeNoSuchDistribution, "msg", nil)
//	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
//	awsClient.On("UpdateDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("GetDistribution", mock.Anything).Return(noError).Once()
//	awsClient.On("DeleteDistribution", mock.Anything).Return(awsErr).Once()
//
//	repo := NewDistributionRepository(awsClient, &test.MockResourceTaggingAPI{}, testCallerRefFn, time.Minute)
//	s.NoError(repo.Delete(Distribution{ID: "id"}))
//}
//
//func (s *DistributionRepositoryTestSuite) Test_baseCacheBehavior_PolicySet() {
//	cb := baseCacheBehavior(
//		Behavior{
//			OriginHost:    "host",
//			PathPattern:   "path",
//			RequestPolicy: "b2884449-e4de-46a7-ac36-70bc7f1ddd6d",
//		},
//	)
//	s.Equal("b2884449-e4de-46a7-ac36-70bc7f1ddd6d", *cb.OriginRequestPolicyId)
//}
//
//func (s *DistributionRepositoryTestSuite) Test_baseCacheBehavior_PolicySetToNone() {
//	cb := baseCacheBehavior(
//		Behavior{
//			OriginHost:    "host",
//			PathPattern:   "path",
//			RequestPolicy: "None",
//		},
//	)
//	s.Nil(cb.OriginRequestPolicyId)
//}
