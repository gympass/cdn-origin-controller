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

package k8s

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

const (
	// CDNGroupAnnotation is the annotation key that represents a group of Ingresses composing a single Distribution
	CDNGroupAnnotation = "cdn-origin-controller.gympass.com/cdn.group"
	// CDNClassAnnotation is the annotation key that represents a class
	CDNClassAnnotation = "cdn-origin-controller.gympass.com/cdn.class"
	// CDNFinalizer is the finalizer to be used in Ingresses managed by the operator
	CDNFinalizer = "cdn-origin-controller.gympass.com/finalizer"

	cfViewerFnAnnotation             = "cdn-origin-controller.gympass.com/cf.viewer-function-arn"
	cfOrigReqPolicyAnnotation        = "cdn-origin-controller.gympass.com/cf.origin-request-policy"
	cfCachePolicyAnnotation          = "cdn-origin-controller.gympass.com/cf.cache-policy"
	cfOrigRespTimeoutAnnotation      = "cdn-origin-controller.gympass.com/cf.origin-response-timeout"
	cfAlternateDomainNamesAnnotation = "cdn-origin-controller.gympass.com/cf.alternate-domain-names"
	cfWebACLARNAnnotation            = "cdn-origin-controller.gympass.com/cf.web-acl-arn"
	cfTagsAnnotation                 = "cdn-origin-controller.gympass.com/cf.tags"
)

// Path represents a path item within an Ingress
type Path struct {
	PathPattern string
	PathType    string
}

// CDNIngress represents an Ingress within the bounded context of cdn-origin-controller
type CDNIngress struct {
	types.NamespacedName
	LoadBalancerHost     string
	Group                string
	Paths                []Path
	ViewerFnARN          string
	OriginReqPolicy      string
	CachePolicy          string
	OriginRespTimeout    int64
	AlternateDomainNames []string
	WebACLARN            string
	IsBeingRemoved       bool
	OriginAccess         string
	Tags                 map[string]string
}

// GetNamespace returns the CDNIngress namespace
func (c CDNIngress) GetNamespace() string {
	return c.Namespace
}

// GetName returns the CDNIngress name
func (c CDNIngress) GetName() string {
	return c.Name
}

// SharedIngressParams represents parameters which might be specified in multiple Ingresses
type SharedIngressParams struct {
	WebACLARN string
}

// NewSharedIngressParams creates a new SharedIngressParams from a slice of CDNIngress
func NewSharedIngressParams(ingresses []CDNIngress) (SharedIngressParams, error) {
	webACLARNs := sets.NewString()
	for _, ing := range ingresses {
		if len(ing.WebACLARN) > 0 {
			webACLARNs.Insert(ing.WebACLARN)
		}
	}

	if len(webACLARNs) > 1 {
		return SharedIngressParams{}, fmt.Errorf("conflicting WAF WebACL ARNs: %v", webACLARNs.List())
	}

	validARN, _ := webACLARNs.PopAny()
	return SharedIngressParams{WebACLARN: validARN}, nil
}

// NewCDNIngressFromV1 creates a new CDNIngress from a v1 Ingress
func NewCDNIngressFromV1(ing *networkingv1.Ingress) (CDNIngress, error) {
	tags, err := tagsAnnotationValue(ing)
	if err != nil {
		return CDNIngress{}, err
	}

	result := CDNIngress{
		NamespacedName: types.NamespacedName{
			Namespace: ing.Namespace,
			Name:      ing.Name,
		},
		Group:                groupAnnotationValue(ing),
		Paths:                pathsV1(ing.Spec.Rules),
		ViewerFnARN:          viewerFnARN(ing),
		OriginReqPolicy:      originReqPolicy(ing),
		CachePolicy:          cachePolicy(ing),
		OriginRespTimeout:    originRespTimeout(ing),
		AlternateDomainNames: alternateDomainNames(ing),
		WebACLARN:            webACLARN(ing),
		IsBeingRemoved:       IsBeingRemovedFromDesiredState(ing),
		Tags:                 tags,
	}

	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		result.LoadBalancerHost = ing.Status.LoadBalancer.Ingress[0].Hostname
	}

	return result, nil
}

func pathsV1(rules []networkingv1.IngressRule) []Path {
	var paths []Path
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := Path{
				PathPattern: p.Path,
				PathType:    string(*p.PathType),
			}
			paths = append(paths, newPath)
		}
	}
	return paths
}

func viewerFnARN(obj client.Object) string {
	return obj.GetAnnotations()[cfViewerFnAnnotation]
}

func originRespTimeout(obj client.Object) int64 {
	val := obj.GetAnnotations()[cfOrigRespTimeoutAnnotation]
	respTimeout, _ := strconv.ParseInt(val, 10, 64)
	return respTimeout
}

func originReqPolicy(obj client.Object) string {
	return obj.GetAnnotations()[cfOrigReqPolicyAnnotation]
}

func cachePolicy(obj client.Object) string {
	return obj.GetAnnotations()[cfCachePolicyAnnotation]
}

func groupAnnotationValue(obj client.Object) string {
	return obj.GetAnnotations()[CDNGroupAnnotation]
}

func tagsAnnotationValue(obj client.Object) (map[string]string, error) {
	tags := map[string]string{}
	tagsData, ok := obj.GetAnnotations()[cfTagsAnnotation]

	if !ok {
		return tags, nil
	}

	err := yaml.Unmarshal([]byte(tagsData), &tags)
	if err != nil {
		return tags, fmt.Errorf("invalid custom tags configuration: %v", err)
	}

	return tags, nil
}

func alternateDomainNames(obj client.Object) (domainNames []string) {
	annValue := obj.GetAnnotations()[cfAlternateDomainNamesAnnotation]

	if len(annValue) > 0 {
		for _, d := range strings.Split(annValue, ",") {
			if !strhelper.Contains(domainNames, d) {
				domainNames = append(domainNames, d)
			}
		}
	}

	return
}

func webACLARN(obj client.Object) string {
	return obj.GetAnnotations()[cfWebACLARNAnnotation]
}
