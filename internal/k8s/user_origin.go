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
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
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
			OriginHost:        o.Host,
			Group:             groupAnnotationValue(obj),
			UnmergedPaths:     o.paths(),
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

type userOrigin struct {
	Host              string                 `yaml:"host"`
	ResponseTimeout   int64                  `yaml:"responseTimeout"`
	Paths             []string               `yaml:"paths"` // deprecated in favor of Behaviors
	Behaviors         []customOriginBehavior `yaml:"behaviors"`
	ViewerFunctionARN string                 `yaml:"viewerFunctionARN"` // deprecated in favor of Behaviors
	RequestPolicy     string                 `yaml:"originRequestPolicy"`
	CachePolicy       string                 `yaml:"cachePolicy"`
	WebACLARN         string                 `yaml:"webACLARN"`
	OriginAccess      string                 `yaml:"originAccess" default:"Public"`
}

type customOriginBehavior struct {
	Path                 string               `yaml:"path"`
	FunctionAssociations FunctionAssociations `yaml:"functionAssociations"`
}

func (o userOrigin) paths() []Path {
	var paths []Path
	for _, p := range o.Paths {
		path := Path{PathPattern: p}
		if len(o.ViewerFunctionARN) > 0 {
			path.FunctionAssociations = newFAFromViewerFunctionARN(o.ViewerFunctionARN)
		}
		paths = append(paths, path)
	}

	for _, b := range o.Behaviors {
		paths = append(paths, Path{
			PathPattern:          b.Path,
			FunctionAssociations: b.FunctionAssociations,
		})
	}

	return paths
}

func (o userOrigin) validate() error {
	if err := o.validateBehaviors(); err != nil {
		return err
	}

	if len(o.Host) == 0 {
		return errors.New("the origin must have a host")
	}

	if len(o.Paths) == 0 && len(o.Behaviors) == 0 {
		return errors.New("the origin must have at least one path or behavior")
	}

	if o.OriginAccess != CFUserOriginAccessPublic && o.OriginAccess != CFUserOriginAccessBucket {
		return fmt.Errorf("the origin must specify a valid originAccess. Valid values: %q, %q",
			CFUserOriginAccessPublic, CFUserOriginAccessBucket)
	}

	return nil
}

func (o userOrigin) validateBehaviors() error {
	for _, b := range o.Behaviors {
		if err := b.FunctionAssociations.Validate(); err != nil {
			return fmt.Errorf("validating behavior function associations: %v", err)
		}

		if strhelper.Contains(o.Paths, b.Path) {
			return fmt.Errorf("same path %q informed in paths (deprecated) and behaviors. Specify it in behaviors only", b.Path)
		}

		if !b.FunctionAssociations.IsEmpty() && len(o.ViewerFunctionARN) > 0 {
			return errors.New("function associations declared in behaviors, but viewerFunctionArn is present (deprecated). " +
				"Configure all functions via behaviors only")
		}
	}
	return nil
}

func userOriginsFromYAML(originsData []byte) ([]userOrigin, error) {
	var origins []userOrigin
	err := yaml.Unmarshal(originsData, &origins)
	if err != nil {
		return nil, err
	}

	for i, o := range origins {
		err = defaults.Set(&o)

		if err != nil {
			return nil, err
		}

		if err := o.validate(); err != nil {
			return nil, fmt.Errorf("validating user origin: %v", err)
		}

		origins[i] = o
	}

	return origins, nil
}
