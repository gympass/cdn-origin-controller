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

package controllers

import (
	"errors"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errUnsupportedKind = errors.New("unsupported kind")

type path struct {
	pathPattern string
	pathType    string
}

type ingressDTO struct {
	host  string
	paths []path
}

func newIngressDTO(obj client.Object) (ingressDTO, error) {
	switch obj := obj.(type) {
	case *networkingv1beta1.Ingress:
		return newIngressDTOV1beta1(obj), nil
	case *networkingv1.Ingress:
		return newIngressDTOV1(obj), nil
	}
	return ingressDTO{}, errUnsupportedKind
}

func newIngressDTOV1beta1(ing *networkingv1beta1.Ingress) ingressDTO {
	return ingressDTO{
		host:  ing.Status.LoadBalancer.Ingress[0].Hostname,
		paths: pathsV1beta1(ing.Spec.Rules),
	}
}

func pathsV1beta1(rules []networkingv1beta1.IngressRule) []path {
	var paths []path
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := path{
				pathPattern: p.Path,
				pathType:    string(*p.PathType),
			}
			paths = append(paths, newPath)
		}
	}
	return paths
}

func newIngressDTOV1(ing *networkingv1.Ingress) ingressDTO {
	return ingressDTO{
		host:  ing.Status.LoadBalancer.Ingress[0].Hostname,
		paths: pathsV1(ing.Spec.Rules),
	}
}

func pathsV1(rules []networkingv1.IngressRule) []path {
	var paths []path
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := path{
				pathPattern: p.Path,
				pathType:    string(*p.PathType),
			}
			paths = append(paths, newPath)
		}
	}
	return paths
}
