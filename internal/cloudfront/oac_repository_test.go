// Copyright (c) 2023 GPBR Participacoes LTDA.
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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/test"
)

type oacListerMock struct {
	mock.Mock
	OACLister
	expectedPages []*awscloudfront.ListOriginAccessControlsOutput
}

func (m *oacListerMock) ListOriginAccessControlsPages(input *awscloudfront.ListOriginAccessControlsInput, fn func(*awscloudfront.ListOriginAccessControlsOutput, bool) bool) error {
	args := m.Called(input, fn)
	for i, expectedPage := range m.expectedPages {
		isLastPage := len(m.expectedPages) == i+1
		shouldContinue := fn(expectedPage, isLastPage)
		if !shouldContinue {
			break
		}
	}
	return args.Error(0)
}

func TestRunOACRepositoryTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &oacRepositorySuite{})
}

type oacRepositorySuite struct {
	suite.Suite
	client *test.MockCloudFrontAPI
	lister *oacListerMock
	cfg    config.Config
}

func (s *oacRepositorySuite) SetupTest() {
	s.client = &test.MockCloudFrontAPI{}
	s.lister = &oacListerMock{}
	s.cfg = config.Config{DeletionEnabled: true}
}

func (s *oacRepositorySuite) TestSync_OACWillBeCreatedAndOtherOACsAlreadyExistShouldReturnNoError() {
	var noError error
	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.lister.expectedPages = []*awscloudfront.ListOriginAccessControlsOutput{
		{
			OriginAccessControlList: &awscloudfront.OriginAccessControlList{Items: []*awscloudfront.OriginAccessControlSummary{
				{
					// we just want this to not match the name, the rest isn't important
					// we want to ensure we go at least through one page
					Id:   aws.String("id"),
					Name: aws.String("another name"),
				},
			}},
		},
	}

	s.client.On("CreateOriginAccessControl", mock.Anything).
		Return(noError)
	s.client.ExpectedCreateOriginAccessControlOutput = &awscloudfront.CreateOriginAccessControlOutput{
		OriginAccessControl: &awscloudfront.OriginAccessControl{
			Id: aws.String("id"),
			OriginAccessControlConfig: &awscloudfront.OriginAccessControlConfig{
				Name:                          aws.String("name"),
				OriginAccessControlOriginType: aws.String(awscloudfront.OriginAccessControlOriginTypesS3),
				SigningBehavior:               aws.String(awscloudfront.OriginAccessControlSigningBehaviorsAlways),
				SigningProtocol:               aws.String(awscloudfront.OriginAccessControlSigningProtocolsSigv4),
			},
		},
	}

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Sync(OAC{
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	})

	s.NoError(err)
	s.Equal(OAC{
		ID:                            "id",
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	}, got)
}

func (s *oacRepositorySuite) TestSync_OACWillBeUpdatedAndShouldReturnNoError() {
	var noError error

	s.client.On("GetOriginAccessControl", mock.Anything).
		Return(noError)
	s.client.ExpectedGetOriginAccessControlOutput = &awscloudfront.GetOriginAccessControlOutput{
		ETag: aws.String("eTag"),
	}

	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.lister.expectedPages = []*awscloudfront.ListOriginAccessControlsOutput{
		{
			OriginAccessControlList: &awscloudfront.OriginAccessControlList{Items: []*awscloudfront.OriginAccessControlSummary{
				{
					Id:   aws.String("id"),
					Name: aws.String("name"), // we just want this to match the name, the rest isn't important
				},
			}},
		},
	}

	s.client.On("UpdateOriginAccessControl", mock.Anything).
		Return(noError)

	s.client.ExpectedUpdateOriginAccessControlOutput = &awscloudfront.UpdateOriginAccessControlOutput{
		OriginAccessControl: &awscloudfront.OriginAccessControl{
			Id: aws.String("id"),
			OriginAccessControlConfig: &awscloudfront.OriginAccessControlConfig{
				Name:                          aws.String("name"),
				OriginAccessControlOriginType: aws.String(awscloudfront.OriginAccessControlOriginTypesS3),
				SigningBehavior:               aws.String(awscloudfront.OriginAccessControlSigningBehaviorsAlways),
				SigningProtocol:               aws.String(awscloudfront.OriginAccessControlSigningProtocolsSigv4),
			},
		},
	}

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Sync(OAC{
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	})

	s.NoError(err)
	s.Equal(OAC{
		ID:                            "id",
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	}, got)
}
func (s *oacRepositorySuite) TestSync_OACFailsToBeFetchedAndShouldReturnError() {
	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(errors.New("some error"))

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Sync(OAC{
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	})

	s.Error(err)
	s.Empty(got)
}

func (s *oacRepositorySuite) TestSync_OACFailsToBeCreatedAndShouldReturnError() {
	var noError error
	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.client.On("CreateOriginAccessControl", mock.Anything).
		Return(errors.New("some error"))

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Sync(OAC{
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	})

	s.Error(err)
	s.Empty(got)
}

func (s *oacRepositorySuite) TestSync_OACFailsToBeUpdatedAndShouldReturnError() {
	var noError error

	s.client.On("GetOriginAccessControl", mock.Anything).
		Return(noError)
	s.client.ExpectedGetOriginAccessControlOutput = &awscloudfront.GetOriginAccessControlOutput{
		ETag: aws.String("eTag"),
	}

	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.lister.expectedPages = []*awscloudfront.ListOriginAccessControlsOutput{
		{
			OriginAccessControlList: &awscloudfront.OriginAccessControlList{Items: []*awscloudfront.OriginAccessControlSummary{
				{
					Id:   aws.String("id"),
					Name: aws.String("name"), // we just want this to match the name, the rest isn't important
				},
			}},
		},
	}

	s.client.On("UpdateOriginAccessControl", mock.Anything).
		Return(errors.New("some error"))

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Sync(OAC{
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	})

	s.Error(err)
	s.Empty(got)
}

func (s *oacRepositorySuite) TestDelete_OACWillBeDeletedAndShouldReturnNoError() {
	var noError error

	s.client.On("GetOriginAccessControl", mock.Anything).
		Return(noError)
	s.client.ExpectedGetOriginAccessControlOutput = &awscloudfront.GetOriginAccessControlOutput{
		ETag: aws.String("eTag"),
	}

	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.lister.expectedPages = []*awscloudfront.ListOriginAccessControlsOutput{
		{
			OriginAccessControlList: &awscloudfront.OriginAccessControlList{Items: []*awscloudfront.OriginAccessControlSummary{
				{
					Id:                            aws.String("id"),
					Name:                          aws.String("name"),
					OriginAccessControlOriginType: aws.String("s3"),
					SigningBehavior:               aws.String("always"),
					SigningProtocol:               aws.String("sigv4"),
				},
			}},
		},
	}

	s.client.On("DeleteOriginAccessControl", mock.Anything).
		Return(noError)

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Delete(OAC{
		Name:       "name",
		OriginName: "OriginName",
	})

	s.NoError(err)
	s.Equal(OAC{
		ID:                            "id",
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	}, got)
}

func (s *oacRepositorySuite) TestDelete_OACDoesNotExistAndShouldReturnNoError() {
	var noError error
	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Delete(OAC{
		Name:       "name",
		OriginName: "OriginName",
	})

	s.NoError(err)
	s.Equal(OAC{}, got)
}

func (s *oacRepositorySuite) TestDelete_OACWasDeletedExternallyAfterFetchingAndShouldReturnNoError() {
	var noError error

	s.client.On("GetOriginAccessControl", mock.Anything).
		Return(noError)
	s.client.ExpectedGetOriginAccessControlOutput = &awscloudfront.GetOriginAccessControlOutput{
		ETag: aws.String("eTag"),
	}

	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.lister.expectedPages = []*awscloudfront.ListOriginAccessControlsOutput{
		{
			OriginAccessControlList: &awscloudfront.OriginAccessControlList{Items: []*awscloudfront.OriginAccessControlSummary{
				{
					Id:                            aws.String("id"),
					Name:                          aws.String("name"),
					OriginAccessControlOriginType: aws.String("s3"),
					SigningBehavior:               aws.String("always"),
					SigningProtocol:               aws.String("sigv4"),
				},
			}},
		},
	}

	s.client.On("DeleteOriginAccessControl", mock.Anything).
		Return(awserr.New(awscloudfront.ErrCodeNoSuchOriginAccessControl, "msg", nil))

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Delete(OAC{
		Name:       "name",
		OriginName: "OriginName",
	})

	s.NoError(err)
	s.Equal(OAC{
		ID:                            "id",
		Name:                          "name",
		OriginName:                    "OriginName",
		OriginAccessControlOriginType: "s3",
		SigningBehavior:               "always",
		SigningProtocol:               "sigv4",
	}, got)
}

func (s *oacRepositorySuite) TestDelete_FailedToGetOACAndShouldReturnError() {
	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(errors.New("some error"))
	got, err := NewOACRepository(s.client, s.lister, s.cfg).Delete(OAC{
		Name:       "name",
		OriginName: "OriginName",
	})

	s.Error(err)
	s.Empty(got)
}

func (s *oacRepositorySuite) TestDelete_FailedToDeleteAndShouldReturnError() {
	var noError error

	s.client.On("GetOriginAccessControl", mock.Anything).
		Return(noError)
	s.client.ExpectedGetOriginAccessControlOutput = &awscloudfront.GetOriginAccessControlOutput{
		ETag: aws.String("eTag"),
	}

	s.lister.On("ListOriginAccessControlsPages", mock.Anything, mock.Anything).
		Return(noError)
	s.lister.expectedPages = []*awscloudfront.ListOriginAccessControlsOutput{
		{
			OriginAccessControlList: &awscloudfront.OriginAccessControlList{Items: []*awscloudfront.OriginAccessControlSummary{
				{
					Id:                            aws.String("id"),
					Name:                          aws.String("name"),
					OriginAccessControlOriginType: aws.String("s3"),
					SigningBehavior:               aws.String("always"),
					SigningProtocol:               aws.String("sigv4"),
				},
			}},
		},
	}

	s.client.On("DeleteOriginAccessControl", mock.Anything).
		Return(errors.New("some error"))

	got, err := NewOACRepository(s.client, s.lister, s.cfg).Delete(OAC{
		Name:       "name",
		OriginName: "OriginName",
	})

	s.Error(err)
	s.Empty(got)
}
