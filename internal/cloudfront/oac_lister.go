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
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	awscloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/cloudfront/cloudfrontiface"
)

// This entire file should be moved to the SDK if they accept a feature request.
// It implements pagination for listing OACs in the same style of other paginators
// that are already implemented in the SDK. For example:
// https://github.com/aws/aws-sdk-go/blob/v1.44.269/service/cloudfront/api.go#L6927

// OACLister lists OACs.
// Using an interface to make it more testable, since otherwise we'd need to create
// fake requests, which can't be mocked because they are not interfaces.
type OACLister interface {
	ListOriginAccessControlsPages(input *awscloudfront.ListOriginAccessControlsInput, fn func(*awscloudfront.ListOriginAccessControlsOutput, bool) bool) error
	ListOriginAccessControlsPagesWithContext(ctx context.Context, input *awscloudfront.ListOriginAccessControlsInput, fn func(*awscloudfront.ListOriginAccessControlsOutput, bool) bool, opts ...request.Option) error
}

type oacLister struct {
	client cloudfrontiface.CloudFrontAPI
}

func NewOACLister(client cloudfrontiface.CloudFrontAPI) OACLister {
	return oacLister{client: client}
}

// ListOriginAccessControlsPages iterates over the pages of an awscloudfront.ListOriginAccessControlsInput operation,
// calling the "fn" function with the response data for each page. To stop
// iterating, return false from the fn function.
//
// See awscloudfront.ListOriginAccessControlsInput method for more information on how to use this operation.
//
// Note: This operation can generate multiple requests to a service.
// Note2: Ideally this would be implemented in the SDK directly, it isn't for now.
//
//	// Example iterating over at most 3 pages of an awscloudfront.ListOriginAccessControls operation.
//	pageNum := 0
//	err := client.ListOriginAccessControlsPages(params,
//	    func(page *awscloudfront.ListOriginAccessControlsOutput, lastPage bool) bool {
//	        pageNum++
//	        fmt.Println(page)
//	        return pageNum <= 3
//	    })
func (l oacLister) ListOriginAccessControlsPages(input *awscloudfront.ListOriginAccessControlsInput, fn func(*awscloudfront.ListOriginAccessControlsOutput, bool) bool) error {
	return l.ListOriginAccessControlsPagesWithContext(aws.BackgroundContext(), input, fn)
}

// ListOriginAccessControlsPagesWithContext same as ListOriginAccessControlsPages except
// it takes a Context and allows setting request options on the pages.
//
// Note: Ideally this would be implemented in the SDK directly, it isn't for now.
//
// The context must be non-nil and will be used for request cancellation. If
// the context is nil a panic will occur. In the future the SDK may create
// sub-contexts for http.Requests. See https://golang.org/pkg/context/
// for more information on using Contexts.
func (l oacLister) ListOriginAccessControlsPagesWithContext(ctx context.Context, input *awscloudfront.ListOriginAccessControlsInput, fn func(*awscloudfront.ListOriginAccessControlsOutput, bool) bool, opts ...request.Option) error {
	p := request.Pagination{
		NewRequest: func() (*request.Request, error) {
			var inCpy *awscloudfront.ListOriginAccessControlsInput
			if input != nil {
				tmp := *input
				inCpy = &tmp
			}
			req, _ := l.client.ListOriginAccessControlsRequest(inCpy)
			req.SetContext(ctx)
			req.ApplyOptions(opts...)
			return req, nil
		},
	}

	for p.Next() {
		if !fn(p.Page().(*awscloudfront.ListOriginAccessControlsOutput), !p.HasNextPage()) {
			break
		}
	}

	return p.Err()
}
