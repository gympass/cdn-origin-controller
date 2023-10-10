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

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CFUserOriginAccessPublic = "Public"
	CFUserOriginAccessBucket = "Bucket"
)

const cfUserOriginsAnnotation = "cdn-origin-controller.gympass.com/cf.user-origins"

func cdnIngressesForUserOrigins(obj client.Object) ([]CDNIngress, error) {
	userOriginsMarkup, ok := obj.GetAnnotations()[cfUserOriginsAnnotation]
	if !ok {
		return nil, nil
	}

	origins, err := userOriginsFromYAML([]byte(userOriginsMarkup))
	if err != nil {
		return nil, fmt.Errorf("ingress %s/%s: parsing user origins data from the %s annotation: %v",
			obj.GetNamespace(), obj.GetName(), cfUserOriginsAnnotation, err)
	}

	var result []CDNIngress
	for _, o := range origins {
		ing := CDNIngress{
			NamespacedName:    types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()},
			LoadBalancerHost:  o.Host,
			Group:             groupAnnotationValue(obj),
			UnmergedPaths:     o.paths(),
			ViewerFnARN:       o.ViewerFunctionARN,
			OriginReqPolicy:   o.RequestPolicy,
			CachePolicy:       o.CachePolicy,
			OriginRespTimeout: o.ResponseTimeout,
			UnmergedWebACLARN: o.WebACLARN,
			OriginAccess:      o.OriginAccess,
		}
		result = append(result, ing)
	}

	return result, nil
}

// TODO in upcoming PRs:
// parse and validate function associations, ensuring:
//   a. it only references paths present in this custom origin
//   b. do not break any existing rules that are already mapped in fa.Validate()

type userOrigin struct {
	Host              string   `yaml:"host"`
	ResponseTimeout   int64    `yaml:"responseTimeout"`
	Paths             []string `yaml:"paths"`
	ViewerFunctionARN string   `yaml:"viewerFunctionARN"`
	RequestPolicy     string   `yaml:"originRequestPolicy"`
	CachePolicy       string   `yaml:"cachePolicy"`
	WebACLARN         string   `yaml:"webACLARN"`
	OriginAccess      string   `yaml:"originAccess" default:"Public"`
}

func (o userOrigin) paths() []Path {
	var paths []Path
	for _, p := range o.Paths {
		paths = append(paths, Path{PathPattern: p})
	}
	return paths
}

func (o userOrigin) isValid() bool {
	return len(o.Host) > 0 && len(o.Paths) > 0 && (o.OriginAccess == CFUserOriginAccessPublic || o.OriginAccess == CFUserOriginAccessBucket)
}

func userOriginsFromYAML(originsData []byte) ([]userOrigin, error) {
	origins := []userOrigin{}
	err := yaml.Unmarshal(originsData, &origins)
	if err != nil {
		return nil, err
	}

	for i, o := range origins {
		err = defaults.Set(&o)

		if err != nil {
			return nil, err
		}

		if !o.isValid() {
			return nil, fmt.Errorf("user origin invalid. Must have at lease one path, must have a host, and origin access must be Public or Bucket: has %d paths and the host is %q", len(o.Paths), o.Host)
		}

		origins[i] = o
	}

	return origins, nil
}
