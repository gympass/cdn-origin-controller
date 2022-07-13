package test

import (
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/stretchr/testify/mock"
)

// MockCloudFrontAPI is mocked cloudfrontiface.CloudFrontAPI to be used during testing
type MockCloudFrontAPI struct {
	mock.Mock
	cloudfrontiface.CloudFrontAPI
	ExpectedGetDistributionConfigOutput      *cloudfront.GetDistributionConfigOutput
	ExpectedUpdateDistributionOutput         *cloudfront.UpdateDistributionOutput
	ExpectedCreateDistributionWithTagsOutput *cloudfront.CreateDistributionWithTagsOutput
	ExpectedTagResourceOutput                *cloudfront.TagResourceOutput
	ExpectedGetDistributionOutput            *cloudfront.GetDistributionOutput
}

func (c *MockCloudFrontAPI) GetDistributionConfig(in *cloudfront.GetDistributionConfigInput) (*cloudfront.GetDistributionConfigOutput, error) {
	args := c.Called(in)
	return c.ExpectedGetDistributionConfigOutput, args.Error(0)
}

func (c *MockCloudFrontAPI) UpdateDistribution(in *cloudfront.UpdateDistributionInput) (*cloudfront.UpdateDistributionOutput, error) {
	args := c.Called(in)
	return c.ExpectedUpdateDistributionOutput, args.Error(0)
}
func (c *MockCloudFrontAPI) GetDistribution(in *cloudfront.GetDistributionInput) (*cloudfront.GetDistributionOutput, error) {
	args := c.Called(in)
	return c.ExpectedGetDistributionOutput, args.Error(0)
}

func (c *MockCloudFrontAPI) CreateDistributionWithTags(in *cloudfront.CreateDistributionWithTagsInput) (*cloudfront.CreateDistributionWithTagsOutput, error) {
	args := c.Called(in)
	return c.ExpectedCreateDistributionWithTagsOutput, args.Error(0)
}

func (c *MockCloudFrontAPI) TagResource(in *cloudfront.TagResourceInput) (*cloudfront.TagResourceOutput, error) {
	args := c.Called(in)
	return c.ExpectedTagResourceOutput, args.Error(0)
}

func (c *MockCloudFrontAPI) DeleteDistribution(in *cloudfront.DeleteDistributionInput) (*cloudfront.DeleteDistributionOutput, error) {
	args := c.Called(in)
	return nil, args.Error(0)
}
