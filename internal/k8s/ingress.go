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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

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
	cfOrigHeadersAnnotation          = "cdn-origin-controller.gympass.com/cf.origin-headers"
)

// Path represents a path item within an Ingress
type Path struct {
	PathPattern          string
	PathType             string
	FunctionAssociations FunctionAssociations
}

// CDNIngress represents an Ingress within the bounded context of cdn-origin-controller
type CDNIngress struct {
	types.NamespacedName
	OriginHost           string
	Group                string
	UnmergedPaths        []Path
	OriginReqPolicy      string
	OriginHeaders        map[string]string
	CachePolicy          string
	OriginRespTimeout    int64
	AlternateDomainNames []string
	UnmergedWebACLARN    string
	IsBeingRemoved       bool
	OriginAccess         string
	Class                CDNClass
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

var (
	errSharedParamsConflictingACL   = errors.New("conflicting WAF WebACL ARNs")
	errSharedParamsConflictingPaths = errors.New("conflicting path configuration")
)

// SharedIngressParams represents parameters which might be specified in multiple Ingresses
type SharedIngressParams struct {
	WebACLARN string
	paths     map[string][]Path // map[originHost][]Path
}

// NewSharedIngressParams creates a new SharedIngressParams from a slice of CDNIngress
func NewSharedIngressParams(ingresses []CDNIngress) (SharedIngressParams, error) {
	acl, err := mergedWebACL(ingresses)
	if err != nil {
		return SharedIngressParams{}, fmt.Errorf("%w: %v", errSharedParamsConflictingACL, err)
	}

	fa, err := mergedPaths(ingresses)
	if err != nil {
		return SharedIngressParams{}, fmt.Errorf("%w: %v", errSharedParamsConflictingPaths, err)
	}

	return SharedIngressParams{
		WebACLARN: acl,
		paths:     fa,
	}, nil
}

func (sp SharedIngressParams) PathsFromOrigin(originHost string) []Path {
	return sp.paths[originHost]
}

func mergedPaths(ingresses []CDNIngress) (map[string][]Path, error) {
	// Let's put it all in a map of Paths to easily check if we already saw this
	// Path before and whether to merge it, but we must avoid checking equality of
	// Paths by also checking Path.FunctionAssociations, because the same Path might
	// be specified in more than one CDNIngress (and should be merged if valid).
	//
	// This is why here we always assign a new Path when indexing the map,
	// instead of just using the input Path, with an empty FunctionAssociations.
	// This way we ensure we only check equality via PathPattern and PathType.
	//
	// We're also going to create a map[originHostname] because this
	// struct will later also need to filter paths by origin, so it comes in handy
	// to filter easily.
	//
	// The resulting map of all of this is a map of Path indexed by ingress name.

	mergedPaths := make(map[string]map[Path]FunctionAssociations)
	for _, ing := range ingresses {
		_, ok := mergedPaths[ing.OriginHost]
		if !ok {
			mergedPaths[ing.OriginHost] = make(map[Path]FunctionAssociations)
		}

		for _, p := range ing.UnmergedPaths {
			pKey := Path{
				PathPattern: p.PathPattern,
				PathType:    p.PathType,
			}
			existingFA, ok := mergedPaths[ing.OriginHost][pKey]
			if !ok {
				mergedPaths[ing.OriginHost][pKey] = p.FunctionAssociations
				continue
			}

			mergedFA, err := existingFA.Merge(p.FunctionAssociations)
			if err != nil {
				return nil, fmt.Errorf("conflicting function associations on %q: %v", p.PathPattern, err)
			}

			mergedPaths[ing.OriginHost][pKey] = mergedFA
		}
	}

	return mapOfOriginHostToPath(mergedPaths), nil
}

func mapOfOriginHostToPath(ingsPathsAndFAs map[string]map[Path]FunctionAssociations) map[string][]Path {
	s := make(map[string][]Path)
	for originHost, pathsAndFAs := range ingsPathsAndFAs {
		for p, fa := range pathsAndFAs {
			path := Path{
				PathPattern:          p.PathPattern,
				PathType:             p.PathType,
				FunctionAssociations: fa,
			}
			s[originHost] = append(s[originHost], path)
		}
	}
	return s
}

func mergedWebACL(ingresses []CDNIngress) (string, error) {
	webACLARNs := sets.NewString()
	for _, ing := range ingresses {
		if len(ing.UnmergedWebACLARN) > 0 {
			webACLARNs.Insert(ing.UnmergedWebACLARN)
		}
	}

	if len(webACLARNs) > 1 {
		return "", fmt.Errorf("more than one ACL specified: %v", webACLARNs.List())
	}

	validARN, _ := webACLARNs.PopAny()
	return validARN, nil
}

// NewCDNIngressFromV1 creates a new CDNIngress from a v1 Ingress
func NewCDNIngressFromV1(ctx context.Context, ing *networkingv1.Ingress, class CDNClass) (CDNIngress, error) {
	tags, err := tagsAnnotationValue(ing)
	if err != nil {
		return CDNIngress{}, err
	}

	paths, err := pathsV1(ctx, ing)
	if err != nil {
		return CDNIngress{}, err
	}

	headers, err := headersV1(ing)
	if err != nil {
		return CDNIngress{}, err
	}

	result := CDNIngress{
		NamespacedName: types.NamespacedName{
			Namespace: ing.Namespace,
			Name:      ing.Name,
		},
		Group:                groupAnnotationValue(ing),
		UnmergedPaths:        paths,
		OriginReqPolicy:      originReqPolicy(ing),
		OriginHeaders:        headers,
		CachePolicy:          cachePolicy(ing),
		OriginRespTimeout:    originRespTimeout(ing),
		AlternateDomainNames: alternateDomainNames(ing),
		UnmergedWebACLARN:    webACLARN(ing),
		IsBeingRemoved:       IsBeingRemovedFromDesiredState(ing),
		Class:                class,
		Tags:                 tags,
		OriginAccess:         CFUserOriginAccessPublic,
	}

	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		result.OriginHost = ing.Status.LoadBalancer.Ingress[0].Hostname
	}

	return result, nil
}

func pathsV1(ctx context.Context, ing *networkingv1.Ingress) ([]Path, error) {
	fa, err := functionAssociations(ing)
	if err != nil {
		return nil, fmt.Errorf("parsing function associations from annotation: %v", err)
	}

	viewerFn := viewerFnARN(ing)
	if len(viewerFn) > 0 && len(fa) > 0 {
		return nil, fmt.Errorf("can't use %q (deprecated) and %q at the same time, prefer %q",
			cfViewerFnAnnotation, cfFunctionAssociationsAnnotation, cfFunctionAssociationsAnnotation)
	}

	if len(viewerFn) > 0 {
		return pathsForViewerFunction(ing, viewerFn), nil
	}
	return pathsForFunctionAssociations(ctx, ing, fa), nil
}

func pathsForViewerFunction(ing *networkingv1.Ingress, fnARN string) []Path {
	rules := ing.Spec.Rules

	var paths []Path
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := Path{
				PathPattern:          p.Path,
				PathType:             string(*p.PathType),
				FunctionAssociations: newFAFromViewerFunctionARN(fnARN),
			}
			paths = append(paths, newPath)
		}
	}

	return paths
}

func pathsForFunctionAssociations(ctx context.Context, ing *networkingv1.Ingress, fa map[string]FunctionAssociations) []Path {
	rules := ing.Spec.Rules

	var paths []Path
	for _, rule := range rules {
		for _, p := range rule.HTTP.Paths {
			newPath := Path{
				PathPattern: p.Path,
				PathType:    string(*p.PathType),
			}

			if err := fa[p.Path].Validate(); err != nil {
				// complain about invalid FAs for now, but don't halt reconciliation of all Ingresses because one of them is bad
				// the bad ingress itself will throw an error when reconciled due to invalid annotation
				log.FromContext(ctx).Error(
					errors.New("invalid function association"),
					"Found invalid function association when calculating desired state",
					"functionAssociation", fa[p.Path],
					"invalidIngress", ing.Namespace+"/"+ing.Name)
			} else {
				newPath.FunctionAssociations = fa[p.Path]
			}

			paths = append(paths, newPath)
		}
	}
	return paths
}

func headersV1(ing *networkingv1.Ingress) (map[string]string, error) {
	val, ok := ing.GetAnnotations()[cfOrigHeadersAnnotation]
	if !ok {
		return nil, nil
	}

	headers, err := parseOriginHeaders(val)
	if err != nil {
		return nil, fmt.Errorf("parsing origin headers from annotation %q: %v", cfViewerFnAnnotation, err)
	}

	return headers, nil
}

func parseOriginHeaders(rawHeaders string) (map[string]string, error) {
	if len(rawHeaders) == 0 {
		return nil, nil
	}

	headers := strings.Split(rawHeaders, ",")

	result := make(map[string]string)
	for _, kv := range headers {
		kvParts := strings.Split(kv, "=")
		if len(kvParts) != 2 || kvParts[0] == "" || kvParts[1] == "" {
			return nil, fmt.Errorf("informed origin header does not follow 'key=value' format: %s", kv)
		}

		result[kvParts[0]] = kvParts[1]
	}

	return result, nil
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
