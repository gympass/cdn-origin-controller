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
	"strconv"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errUnsupportedKind = errors.New("unsupported kind")

type path struct {
	pathPattern string
	pathType    string
}

type ingressParams struct {
	loadBalancer      string
	hosts             []string
	group             string
	paths             []path
	viewerFnARN       string
	originRespTimeout int64
}

func newIngressParams(obj client.Object) (ingressParams, error) {
	switch obj := obj.(type) {
	case *networkingv1beta1.Ingress:
		return newIngressV1beta1(obj), nil
	case *networkingv1.Ingress:
		return newIngressV1(obj), nil
	}
	return ingressParams{}, errUnsupportedKind
}

func newIngressV1beta1(ing *networkingv1beta1.Ingress) ingressParams {
	hosts, paths := hostsAndPathsV1beta1(ing.Spec.Rules)
	return ingressParams{
		loadBalancer:      ing.Status.LoadBalancer.Ingress[0].Hostname,
		group:             groupAnnotationValue(ing),
		hosts:             hosts,
		paths:             paths,
		viewerFnARN:       viewerFnARN(ing),
		originRespTimeout: originRespTimeout(ing),
	}
}

func hostsAndPathsV1beta1(rules []networkingv1beta1.IngressRule) (hosts []string, paths []path) {
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := path{
				pathPattern: p.Path,
				pathType:    string(*p.PathType),
			}
			paths = append(paths, newPath)
		}
		hosts = append(hosts, rule.Host)
	}
	return
}

func newIngressV1(ing *networkingv1.Ingress) ingressParams {
	hosts, paths := hostsAndPathsV1(ing.Spec.Rules)
	return ingressParams{
		loadBalancer:      ing.Status.LoadBalancer.Ingress[0].Hostname,
		group:             groupAnnotationValue(ing),
		hosts:             hosts,
		paths:             paths,
		viewerFnARN:       viewerFnARN(ing),
		originRespTimeout: originRespTimeout(ing),
	}
}

func hostsAndPathsV1(rules []networkingv1.IngressRule) (hosts []string, paths []path) {
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := path{
				pathPattern: p.Path,
				pathType:    string(*p.PathType),
			}
			paths = append(paths, newPath)
		}
		hosts = append(hosts, rule.Host)
	}
	return
}

func viewerFnARN(obj client.Object) string {
	return obj.GetAnnotations()[cfViewerFnAnnotation]
}

func originRespTimeout(obj client.Object) int64 {
	val := obj.GetAnnotations()[cfOrigRespTimeoutAnnotation]
	respTimeout, _ := strconv.ParseInt(val, 10, 64)
	return respTimeout
}

func groupAnnotationValue(obj client.Object) string {
	return obj.GetAnnotations()[cdnGroupAnnotation]
}
