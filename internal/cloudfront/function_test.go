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
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/stretchr/testify/suite"

	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

func TestRunCDNIngressTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, &functionTestSuite{})
}

type functionTestSuite struct {
	suite.Suite
}

func (s *functionTestSuite) TestNewFunctions() {
	fa := k8s.FunctionAssociations{
		ViewerRequest: &k8s.ViewerRequestFunction{
			ViewerFunction: k8s.ViewerFunction{
				ARN:          "viewer-request-arn",
				FunctionType: k8s.FunctionTypeEdge,
			},
			IncludeBody: true,
		},
		ViewerResponse: &k8s.ViewerFunction{
			ARN:          "viewer-response-arn",
			FunctionType: k8s.FunctionTypeCloudfront,
		},
		OriginRequest: &k8s.OriginRequestFunction{
			OriginFunction: k8s.OriginFunction{
				ARN: "origin-request-arn",
			},
			IncludeBody: false,
		},
		OriginResponse: &k8s.OriginFunction{
			ARN: "origin-response-arn",
		},
	}

	expected := []Function{
		requestEdgeFunction{
			bodyIncluderFunction: bodyIncluderFunction{
				basicFunction: basicFunction{
					arn:       "viewer-request-arn",
					eventType: cloudfront.EventTypeViewerRequest,
				},
				includeBody: true,
			},
		},
		responseCloudfrontFunction{
			basicFunction: basicFunction{
				arn:       "viewer-response-arn",
				eventType: cloudfront.EventTypeViewerResponse,
			},
		},
		requestEdgeFunction{
			bodyIncluderFunction: bodyIncluderFunction{
				basicFunction: basicFunction{
					arn:       "origin-request-arn",
					eventType: cloudfront.EventTypeOriginRequest,
				},
				includeBody: false,
			},
		},
		responseEdgeFunction{
			basicFunction: basicFunction{
				arn:       "origin-response-arn",
				eventType: cloudfront.EventTypeOriginResponse,
			},
		},
	}

	got := NewFunctions(fa)

	s.Equal(expected, got)
}
