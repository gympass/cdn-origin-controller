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

package test

import (
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
	"github.com/stretchr/testify/mock"
)

// MockCloudFrontAPI is a mocked cloudfrontiface.CloudFrontAPI to be used during testing
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
