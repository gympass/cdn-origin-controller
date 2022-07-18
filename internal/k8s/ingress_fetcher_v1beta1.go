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

package k8s

import (
	"context"
	"fmt"

	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ingFetcherV1Beta1 struct {
	k8sClient client.Client
}

// NewIngressFetcherV1Beta1 creates an IngressFetcher that works with v1beta1 Ingreses
func NewIngressFetcherV1Beta1(k8sClient client.Client) IngressFetcher {
	return ingFetcherV1Beta1{k8sClient: k8sClient}
}

func (i ingFetcherV1Beta1) Fetch(ctx context.Context, name, namespace string) (CDNIngress, error) {
	ing := &networkingv1beta1.Ingress{}
	key := client.ObjectKey{Name: name, Namespace: namespace}

	if err := i.k8sClient.Get(ctx, key, ing); err != nil {
		return CDNIngress{}, err
	}

	return NewCDNIngressFromV1beta1(ing), nil
}

func (i ingFetcherV1Beta1) FetchBy(ctx context.Context, predicate func(CDNIngress) bool) ([]CDNIngress, error) {
	list := &networkingv1beta1.IngressList{}
	if err := i.k8sClient.List(ctx, list); err != nil {
		return nil, fmt.Errorf("listing Ingresses: %v", err)
	}

	var result []CDNIngress
	for _, k8sIng := range list.Items {
		ing := NewCDNIngressFromV1beta1(&k8sIng)
		if predicate(ing) {
			result = append(result, ing)
		}
	}
	return result, nil
}
