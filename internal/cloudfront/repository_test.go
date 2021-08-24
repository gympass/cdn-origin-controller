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

var sslProtocols = []*string{
	aws.String("SSLv3"),
	aws.String("TLSv1"),
	aws.String("TLSv1.1"),
	aws.String("TLSv1.2"),
}

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

func TestRunOriginRepositoryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &OriginRepositoryTestSuite{})
}

type OriginRepositoryTestSuite struct {
	suite.Suite
}

func (s *OriginRepositoryTestSuite) TestOriginRepository_Save_CantFetchDistribution() {
	awsClient := &awsClientMock{}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(errors.New("mock err")).Once()

	repo := cloudfront.NewOriginRepository(awsClient)
	s.Error(repo.Save("mock id", cloudfront.Origin{}))
}

func (s *OriginRepositoryTestSuite) TestOriginRepository_Save_CantUpdateDistribution() {
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

	repo := cloudfront.NewOriginRepository(awsClient)
	s.Error(repo.Save("mock id", cloudfront.Origin{}))
}

func (s *OriginRepositoryTestSuite) TestOriginRepository_Save_OriginDoesNotExistYet() {
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(0)},
		},
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
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
				},
				Quantity: aws.Int64(1),
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

	repo := cloudfront.NewOriginRepository(awsClient)
	s.NoError(repo.Save("mock id", cloudfront.Origin{Host: "origin"}))
}

func (s *OriginRepositoryTestSuite) TestOriginRepository_Save_OriginAlreadyExists() {
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
				},
				Quantity: aws.Int64(1),
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

	repo := cloudfront.NewOriginRepository(awsClient)
	s.NoError(repo.Save("mock id", cloudfront.Origin{Host: "origin"}))
}

func (s *OriginRepositoryTestSuite) TestOriginRepository_Save_BehaviorDoesNotExistYet() {
	lowerPrecedenceExistingBehavior := &awscloudfront.CacheBehavior{PathPattern: aws.String("/low/precedence/path")}
	expectedDistributionConfigOutput := &awscloudfront.GetDistributionConfigOutput{
		ETag: aws.String(""),
		DistributionConfig: &awscloudfront.DistributionConfig{
			Origins:        &awscloudfront.Origins{Quantity: aws.Int64(0)},
			CacheBehaviors: &awscloudfront.CacheBehaviors{Quantity: aws.Int64(1), Items: []*awscloudfront.CacheBehavior{lowerPrecedenceExistingBehavior}},
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
		PathPattern:                aws.String("/longer/path/with/higher/precedence"),
		SmoothStreaming:            aws.Bool(false),
		TargetOriginId:             aws.String("origin"),
		ViewerProtocolPolicy:       aws.String(awscloudfront.ViewerProtocolPolicyRedirectToHttps),
	}

	expectedUpdateDistributionInput := &awscloudfront.UpdateDistributionInput{
		DistributionConfig: &awscloudfront.DistributionConfig{
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
				},
				Quantity: aws.Int64(1),
			},
			CacheBehaviors: &awscloudfront.CacheBehaviors{
				Items: []*awscloudfront.CacheBehavior{
					expectedNewCacheBehavior,
					lowerPrecedenceExistingBehavior,
				},
				Quantity: aws.Int64(2),
			},
		},
		Id:      aws.String("mock id"),
		IfMatch: aws.String(""),
	}

	var noError error
	awsClient := &awsClientMock{expectedGetDistributionConfigOutput: expectedDistributionConfigOutput}
	awsClient.On("GetDistributionConfig", mock.Anything).Return(noError).Once()
	awsClient.On("UpdateDistribution", expectedUpdateDistributionInput).Return(noError).Once()

	repo := cloudfront.NewOriginRepository(awsClient)
	s.NoError(repo.Save("mock id", cloudfront.Origin{Host: "origin", Behaviors: []cloudfront.Behavior{{PathPattern: "/longer/path/with/higher/precedence"}}}))
}

func (s *OriginRepositoryTestSuite) TestOriginRepository_Save_BehaviorAlreadyExists() {
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
			Origins: existingOrigins,
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

	repo := cloudfront.NewOriginRepository(awsClient)
	s.NoError(repo.Save("mock id", cloudfront.Origin{Host: "origin", Behaviors: []cloudfront.Behavior{{PathPattern: "/*"}}}))
}
