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
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Gympass/cdn-origin-controller/api/v1alpha1"
)

type CDNClassFetcher interface {
	FetchByName(ctx context.Context, name string) (CDNClass, error)
}

type cdnClassFetcher struct {
	k8sClient client.Client
}

func NewCDNClassFetcher(client client.Client) CDNClassFetcher {
	return cdnClassFetcher{k8sClient: client}
}

func (c cdnClassFetcher) FetchByName(ctx context.Context, name string) (CDNClass, error) {
	k8sClass := &v1alpha1.CDNClass{}
	err := c.k8sClient.Get(ctx, types.NamespacedName{Name: name}, k8sClass)

	if err != nil {
		return CDNClass{}, err
	}

	return CDNClass{
		HostedZoneID:  k8sClass.Spec.HostedZoneID,
		CreateAlias:   k8sClass.Spec.CreateAlias,
		TXTOwnerValue: k8sClass.Spec.TXTOwnerValue,
	}, err
}
