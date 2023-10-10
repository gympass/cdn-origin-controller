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
	"github.com/aws/aws-sdk-go/service/cloudfront"

	"github.com/Gympass/cdn-origin-controller/internal/k8s"
)

type Function interface {
	ARN() string
	Type() k8s.FunctionType
	EventType() string
}

type BodyIncluderFunction interface {
	Function
	IncludeBody() bool
}

func NewFunctions(fa k8s.FunctionAssociations) []Function {
	var functions []Function
	if fa.ViewerRequest != nil {
		var fn Function = newRequestCloudfrontFunction(fa.ViewerRequest.ARN, cloudfront.EventTypeViewerRequest)
		if fa.ViewerRequest.FunctionType == k8s.FunctionTypeEdge {
			fn = newRequestEdgeFunction(fa.ViewerRequest.ARN, cloudfront.EventTypeViewerRequest, fa.ViewerRequest.IncludeBody)
		}
		functions = append(functions, fn)
	}
	if fa.ViewerResponse != nil {
		var fn Function = newResponseCloudfrontFunction(fa.ViewerResponse.ARN, cloudfront.EventTypeViewerResponse)
		if fa.ViewerResponse.FunctionType == k8s.FunctionTypeEdge {
			fn = newResponseEdgeFunction(fa.ViewerResponse.ARN, cloudfront.EventTypeViewerResponse)
		}
		functions = append(functions, fn)
	}
	if fa.OriginRequest != nil {
		fn := newRequestEdgeFunction(fa.OriginRequest.ARN, cloudfront.EventTypeOriginRequest, fa.OriginRequest.IncludeBody)
		functions = append(functions, fn)
	}
	if fa.OriginResponse != nil {
		fn := newResponseEdgeFunction(fa.OriginResponse.ARN, cloudfront.EventTypeOriginResponse)
		functions = append(functions, fn)
	}

	return functions
}

var _ Function = requestCloudfrontFunction{}

type requestCloudfrontFunction struct {
	basicFunction
}

func newRequestCloudfrontFunction(arn string, eventType string) requestCloudfrontFunction {
	return requestCloudfrontFunction{
		basicFunction: newBasicFunction(arn, eventType),
	}
}

func (requestCloudfrontFunction) Type() k8s.FunctionType {
	return k8s.FunctionTypeCloudfront
}

var _ BodyIncluderFunction = requestEdgeFunction{}

type requestEdgeFunction struct {
	bodyIncluderFunction
}

func newRequestEdgeFunction(arn string, eventType string, includeBody bool) requestEdgeFunction {
	return requestEdgeFunction{
		bodyIncluderFunction: newBodyIncluderFunction(arn, eventType, includeBody),
	}
}

func (requestEdgeFunction) Type() k8s.FunctionType {
	return k8s.FunctionTypeEdge
}

var _ Function = responseCloudfrontFunction{}

type responseCloudfrontFunction struct {
	basicFunction
}

func newResponseCloudfrontFunction(arn string, eventType string) responseCloudfrontFunction {
	return responseCloudfrontFunction{
		basicFunction: newBasicFunction(arn, eventType),
	}
}

func (f responseCloudfrontFunction) Type() k8s.FunctionType {
	return k8s.FunctionTypeCloudfront
}

var _ Function = responseEdgeFunction{}

type responseEdgeFunction struct {
	basicFunction
}

func newResponseEdgeFunction(arn string, eventType string) responseEdgeFunction {
	return responseEdgeFunction{
		basicFunction: newBasicFunction(arn, eventType),
	}
}

func (responseEdgeFunction) Type() k8s.FunctionType {
	return k8s.FunctionTypeEdge
}

type bodyIncluderFunction struct {
	basicFunction
	includeBody bool
}

func newBodyIncluderFunction(arn string, eventType string, includeBody bool) bodyIncluderFunction {
	return bodyIncluderFunction{
		basicFunction: newBasicFunction(arn, eventType),
		includeBody:   includeBody,
	}
}

func (f bodyIncluderFunction) IncludeBody() bool {
	return f.includeBody
}

type basicFunction struct {
	arn       string
	eventType string
}

func newBasicFunction(arn string, eventType string) basicFunction {
	return basicFunction{
		arn:       arn,
		eventType: eventType,
	}
}

func (f basicFunction) ARN() string {
	return f.arn
}

func (f basicFunction) EventType() string {
	return f.eventType
}
