package test

import (
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/stretchr/testify/mock"
)

// MockResourceTaggingAPI is mocked resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI to be used during testing
type MockResourceTaggingAPI struct {
	mock.Mock
	resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	ExpectedGetResourcesOutput *resourcegroupstaggingapi.GetResourcesOutput
}

func (m *MockResourceTaggingAPI) GetResources(in *resourcegroupstaggingapi.GetResourcesInput) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	args := m.Called(in)
	return m.ExpectedGetResourcesOutput, args.Error(0)
}
