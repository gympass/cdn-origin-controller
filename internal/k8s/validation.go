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
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

func ValidateIngressFunctionAssociations(ing *networkingv1.Ingress) error {
	allFAs, err := functionAssociations(ing)
	if err != nil {
		return fmt.Errorf("parsing function associations: %v", err)
	}

	if allFAs == nil {
		return nil
	}

	if len(viewerFnARN(ing)) > 0 {
		return fmt.Errorf("can't use %q (deprecated) and %q at the same time, prefer %q",
			cfViewerFnAnnotation, cfFunctionAssociationsAnnotation, cfFunctionAssociationsAnnotation)
	}

	ingPaths := ingressPaths(ing)
	for faPath, fa := range allFAs {
		if err := fa.Validate(); err != nil {
			return fmt.Errorf("invalid function association at path %q: %v", faPath, err)
		}

		if !strhelper.Contains(ingPaths, faPath) {
			return fmt.Errorf("function associations references a path %q that is not part of the Ingress' paths %v", faPath, ingPaths)
		}
	}

	return nil
}

func ingressPaths(ing *networkingv1.Ingress) []string {
	var paths []string
	for _, r := range ing.Spec.Rules {
		for _, p := range r.HTTP.Paths {
			paths = append(paths, p.Path)
		}
	}
	return paths
}
