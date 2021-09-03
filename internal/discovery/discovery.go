// Copyright (c) 2021 GPBR Participacoes LTDA.
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

package discovery

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/discovery"
)

const ingressKind = "Ingress"

func HasV1beta1Ingress(client discovery.DiscoveryInterface) (bool, error) {
	gv := networkingv1beta1.SchemeGroupVersion.String()
	return groupVersionHasIngress(gv, client)
}

func HasV1Ingress(client discovery.DiscoveryInterface) (bool, error) {
	gv := networkingv1.SchemeGroupVersion.String()
	return groupVersionHasIngress(gv, client)
}

func groupVersionHasIngress(gv string, client discovery.DiscoveryInterface) (bool, error) {
	resourceList, err := client.ServerResourcesForGroupVersion(gv)
	if err != nil {
		return false, fmt.Errorf("fetching resources for %s: %v", gv, err)
	}

	for _, r := range resourceList.APIResources {
		if r.Kind == ingressKind {
			return true, nil
		}
	}

	return false, nil
}
