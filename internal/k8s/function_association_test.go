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

package k8s

import (
	"testing"

	"github.com/stretchr/testify/suite"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunFunctionAssociationsSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &functionAssociationsTestSuite{})
}

type functionAssociationsTestSuite struct {
	suite.Suite
}

func (s *functionAssociationsTestSuite) Test_functionAssociations_Valid() {
	yaml := `
/foo/*:
  viewerRequest:
    arn: arn:aws:cloudfront::000000000000:function/test-function-associations
    functionType: cloudfront
  viewerResponse:
    arn: arn:aws:cloudfront::000000000000:function/test-function-associations
    functionType: cloudfront
  originRequest:
    arn: arn:aws:lambda:us-east-1:000000000000:function:test-function-associations
    includeBody: true
  originResponse:
    arn: arn:aws:lambda:us-east-1:000000000000:function:test-function-associations
/bar:
  viewerRequest:
    arn: arn:aws:cloudfront::000000000000:function/test-function-associations
    functionType: cloudfront
`
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				cfFunctionAssociationsAnnotation: yaml,
			},
		},
	}

	got, err := functionAssociations(ing)

	s.NoError(err)
	s.Equal(map[string]FunctionAssociations{
		"/foo/*": {
			ViewerRequest: &ViewerRequestFunction{
				ViewerFunction: ViewerFunction{
					ARN:          "arn:aws:cloudfront::000000000000:function/test-function-associations",
					FunctionType: FunctionTypeCloudfront,
				},
			},
			ViewerResponse: &ViewerFunction{
				ARN:          "arn:aws:cloudfront::000000000000:function/test-function-associations",
				FunctionType: FunctionTypeCloudfront,
			},
			OriginRequest: &OriginRequestFunction{
				OriginFunction: OriginFunction{
					ARN: "arn:aws:lambda:us-east-1:000000000000:function:test-function-associations",
				},
				IncludeBody: true,
			},
			OriginResponse: &OriginFunction{
				ARN: "arn:aws:lambda:us-east-1:000000000000:function:test-function-associations",
			},
		},
		"/bar": {
			ViewerRequest: &ViewerRequestFunction{
				ViewerFunction: ViewerFunction{
					ARN:          "arn:aws:cloudfront::000000000000:function/test-function-associations",
					FunctionType: FunctionTypeCloudfront,
				},
			},
		},
	}, got)
}

func (s *functionAssociationsTestSuite) Test_functionAssociations_InvalidYAML() {
	yaml := `	foo`
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				cfFunctionAssociationsAnnotation: yaml,
			},
		},
	}

	got, err := functionAssociations(ing)

	s.Error(err)
	s.Nil(got)
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Validate_ValidFunctions() {
	testCases := []struct {
		name  string
		input FunctionAssociations
	}{
		{
			name:  "nil functions are valid",
			input: FunctionAssociations{},
		},
		{
			name: "properly filled functions are valid",
			input: FunctionAssociations{
				ViewerRequest: &ViewerRequestFunction{
					IncludeBody: false,
					ViewerFunction: ViewerFunction{
						ARN:          "some-arn",
						FunctionType: FunctionTypeCloudfront,
					}},
				ViewerResponse: &ViewerFunction{
					ARN:          "some-arn",
					FunctionType: FunctionTypeCloudfront,
				},
				OriginRequest: &OriginRequestFunction{
					IncludeBody: true,
					OriginFunction: OriginFunction{
						ARN: "some-other-arn",
					}},
				OriginResponse: &OriginFunction{
					ARN: "some-arn",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.NoErrorf(tc.input.Validate(), "test case: %s", tc.name)
	}
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Validate_DifferentFunctionTypesOnViewerIsInvalid() {
	fa := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{ViewerFunction: ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeCloudfront,
		}},
		ViewerResponse: &ViewerFunction{
			ARN:          "another-arn",
			FunctionType: FunctionTypeEdge,
		},
	}

	s.Error(fa.Validate())
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Validate_IncludeBodyForCloudfrontFunctionIsInvalid() {
	fa := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
			IncludeBody: true,
		},
	}

	s.Error(fa.Validate())
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Validate_IncludeBodyForEdgeFunctionIsValid() {
	fa := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeEdge,
			},
			IncludeBody: true,
		},
		OriginRequest: &OriginRequestFunction{
			OriginFunction: OriginFunction{
				ARN: "another-arn",
			},
			IncludeBody: true,
		},
	}

	s.NoError(fa.Validate())
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Validate_InvalidFunctionType() {
	testCases := []struct {
		name  string
		input FunctionAssociations
	}{
		{
			name: "on viewerRequest",
			input: FunctionAssociations{
				ViewerRequest: &ViewerRequestFunction{ViewerFunction: ViewerFunction{
					ARN:          "some-arn",
					FunctionType: "not-a-valid-function-type",
				}},
			},
		},
		{
			name: "on viewerResponse",
			input: FunctionAssociations{
				ViewerResponse: &ViewerFunction{
					ARN:          "some-arn",
					FunctionType: "not-a-valid-function-type",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Errorf(tc.input.Validate(), "test case: %v", tc.name)
	}
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Validate_NoARNIsInvalid() {
	testCases := []struct {
		name  string
		input FunctionAssociations
	}{
		{
			name: "on viewerRequest",
			input: FunctionAssociations{
				ViewerRequest: &ViewerRequestFunction{
					ViewerFunction: ViewerFunction{
						FunctionType: FunctionTypeEdge,
					},
					IncludeBody: true,
				},
			},
		},
		{
			name: "on viewerResponse",
			input: FunctionAssociations{
				ViewerResponse: &ViewerFunction{
					FunctionType: FunctionTypeCloudfront,
				},
			},
		},
		{
			name: "on originRequest",
			input: FunctionAssociations{
				OriginRequest: &OriginRequestFunction{},
			},
		},
		{
			name: "on originResponse",
			input: FunctionAssociations{
				OriginResponse: &OriginFunction{},
			},
		},
	}

	for _, tc := range testCases {
		s.Errorf(tc.input.Validate(), "test case: %v", tc.name)
	}
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_ChangingMergedFADoesNotChangeOriginalOrInput() {
	original := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		},
		ViewerResponse: &ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeCloudfront,
		},
	}

	input := FunctionAssociations{
		OriginRequest: &OriginRequestFunction{
			OriginFunction: OriginFunction{
				ARN: "some-arn",
			},
			IncludeBody: true,
		},
	}

	merged, err := original.Merge(input)
	s.NoError(err)

	merged.ViewerRequest.ViewerFunction.ARN = "some-other-arn"
	merged.ViewerResponse.ARN = "some-other-arn"

	s.Equal(FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		},
		ViewerResponse: &ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeCloudfront,
		},
	}, original)
	s.Equal(FunctionAssociations{
		OriginRequest: &OriginRequestFunction{
			OriginFunction: OriginFunction{
				ARN: "some-arn",
			},
			IncludeBody: true,
		},
	}, input)
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_OriginalIsEmptyAndAbleToMerge() {
	originalFa := FunctionAssociations{}
	mergingFa := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		},
		ViewerResponse: &ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeCloudfront,
		},
		OriginRequest: &OriginRequestFunction{
			IncludeBody: true,
			OriginFunction: OriginFunction{
				ARN: "another-arn",
			},
		},
		OriginResponse: &OriginFunction{
			ARN: "another-arn",
		},
	}

	got, err := originalFa.Merge(mergingFa)

	s.NoError(err)

	s.Equal("some-arn", got.ViewerRequest.ViewerFunction.ARN)
	s.Equal(FunctionTypeCloudfront, got.ViewerRequest.ViewerFunction.FunctionType)

	s.Equal("some-arn", got.ViewerResponse.ARN)
	s.Equal(FunctionTypeCloudfront, got.ViewerResponse.FunctionType)

	s.Equal("another-arn", got.OriginRequest.OriginFunction.ARN)
	s.True(got.OriginRequest.IncludeBody)

	s.Equal("another-arn", got.OriginResponse.ARN)
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_MergingIsEmptyAndAbleToMerge() {
	originalFa := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		},
		ViewerResponse: &ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeCloudfront,
		},
		OriginRequest: &OriginRequestFunction{
			IncludeBody: true,
			OriginFunction: OriginFunction{
				ARN: "another-arn",
			},
		},
		OriginResponse: &OriginFunction{
			ARN: "another-arn",
		},
	}
	mergingFa := FunctionAssociations{}

	got, err := originalFa.Merge(mergingFa)

	s.NoError(err)

	s.Equal("some-arn", got.ViewerRequest.ViewerFunction.ARN)
	s.Equal(FunctionTypeCloudfront, got.ViewerRequest.ViewerFunction.FunctionType)

	s.Equal("some-arn", got.ViewerResponse.ARN)
	s.Equal(FunctionTypeCloudfront, got.ViewerResponse.FunctionType)

	s.Equal("another-arn", got.OriginRequest.OriginFunction.ARN)
	s.True(got.OriginRequest.IncludeBody)

	s.Equal("another-arn", got.OriginResponse.ARN)
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_NeitherIsEmptyButAreAbleToMerge() {
	originalFa := FunctionAssociations{
		ViewerRequest: &ViewerRequestFunction{
			ViewerFunction: ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		},
		ViewerResponse: &ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeCloudfront,
		},
	}
	mergingFa := FunctionAssociations{
		OriginRequest: &OriginRequestFunction{
			IncludeBody: true,
			OriginFunction: OriginFunction{
				ARN: "another-arn",
			},
		},
		OriginResponse: &OriginFunction{
			ARN: "another-arn",
		},
	}

	got, err := originalFa.Merge(mergingFa)

	s.NoError(err)

	s.Equal("some-arn", got.ViewerRequest.ViewerFunction.ARN)
	s.Equal(FunctionTypeCloudfront, got.ViewerRequest.ViewerFunction.FunctionType)

	s.Equal("some-arn", got.ViewerResponse.ARN)
	s.Equal(FunctionTypeCloudfront, got.ViewerResponse.FunctionType)

	s.Equal("another-arn", got.OriginRequest.OriginFunction.ARN)
	s.True(got.OriginRequest.IncludeBody)

	s.Equal("another-arn", got.OriginResponse.ARN)
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_ConflictingViewerRequest() {
	originalFn := &ViewerRequestFunction{
		ViewerFunction: ViewerFunction{
			ARN:          "some-arn",
			FunctionType: FunctionTypeEdge,
		},
		IncludeBody: true,
	}

	testCases := []struct {
		name string
		fn   *ViewerRequestFunction
	}{
		{
			name: "ARN is different",
			fn: &ViewerRequestFunction{
				ViewerFunction: ViewerFunction{
					ARN:          "some-other-arn",
					FunctionType: FunctionTypeEdge,
				},
				IncludeBody: true,
			},
		},
		{
			name: "FunctionType is different",
			fn: &ViewerRequestFunction{
				ViewerFunction: ViewerFunction{
					ARN:          "some-arn",
					FunctionType: FunctionTypeCloudfront,
				},
				IncludeBody: true,
			},
		},
		{
			name: "IncludeBody is different",
			fn: &ViewerRequestFunction{
				ViewerFunction: ViewerFunction{
					ARN:          "some-arn",
					FunctionType: FunctionTypeEdge,
				},
				IncludeBody: false,
			},
		},
	}

	for _, tc := range testCases {
		originalFa := FunctionAssociations{
			ViewerRequest: originalFn,
		}
		mergingFa := FunctionAssociations{
			ViewerRequest: tc.fn,
		}

		merged, err := originalFa.Merge(mergingFa)

		s.Errorf(err, "test case: %s", tc.name)
		s.Emptyf(merged, "test case: %s", tc.name)
	}
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_ConflictingViewerResponse() {
	originalFn := &ViewerFunction{
		ARN:          "some-arn",
		FunctionType: FunctionTypeCloudfront,
	}

	testCases := []struct {
		name string
		fn   *ViewerFunction
	}{
		{
			name: "ARN is different",
			fn: &ViewerFunction{
				ARN:          "some-other-arn",
				FunctionType: FunctionTypeCloudfront,
			},
		},
		{
			name: "FunctionType is different",
			fn: &ViewerFunction{
				ARN:          "some-arn",
				FunctionType: FunctionTypeEdge,
			},
		},
	}

	for _, tc := range testCases {
		originalFa := FunctionAssociations{
			ViewerResponse: originalFn,
		}
		mergingFa := FunctionAssociations{
			ViewerResponse: tc.fn,
		}

		merged, err := originalFa.Merge(mergingFa)

		s.Errorf(err, "test case: %s", tc.name)
		s.Emptyf(merged, "test case: %s", tc.name)
	}
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_ConflictingOriginRequest() {
	originalFn := &OriginRequestFunction{
		OriginFunction: OriginFunction{
			ARN: "some-arn",
		},
		IncludeBody: true,
	}

	testCases := []struct {
		name string
		fn   *OriginRequestFunction
	}{
		{
			name: "ARN is different",
			fn: &OriginRequestFunction{
				OriginFunction: OriginFunction{
					ARN: "some-other-arn",
				},
				IncludeBody: true,
			},
		},
		{
			name: "IncludeBody is different",
			fn: &OriginRequestFunction{
				OriginFunction: OriginFunction{
					ARN: "some-arn",
				},
				IncludeBody: false,
			},
		},
	}

	for _, tc := range testCases {
		originalFa := FunctionAssociations{
			OriginRequest: originalFn,
		}
		mergingFa := FunctionAssociations{
			OriginRequest: tc.fn,
		}

		merged, err := originalFa.Merge(mergingFa)

		s.Errorf(err, "test case: %s", tc.name)
		s.Emptyf(merged, "test case: %s", tc.name)
	}
}

func (s *functionAssociationsTestSuite) TestFunctionAssociations_Merge_ConflictingOriginResponse() {
	originalFa := FunctionAssociations{
		OriginResponse: &OriginFunction{
			ARN: "some-arn",
		},
	}

	mergingFa := FunctionAssociations{
		OriginResponse: &OriginFunction{
			ARN: "some-other-arn",
		},
	}

	merged, err := originalFa.Merge(mergingFa)

	s.Error(err)
	s.Empty(merged)
}
