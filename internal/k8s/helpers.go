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
	"k8s.io/api/networking/v1"
	"k8s.io/api/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

func CDNClassAnnotationValue(object client.Object) string {
	return object.GetAnnotations()[CDNClassAnnotation]
}

func CDNClassMatches(candidate string) bool {
	return candidate == config.CDNClass()
}

func HasFinalizer(object client.Object) bool {
	return strhelper.Contains(object.GetFinalizers(), CDNFinalizer)
}

func AddFinalizer(object client.Object) {
	controllerutil.AddFinalizer(object, CDNFinalizer)
}

func RemoveFinalizer(object client.Object) {
	controllerutil.RemoveFinalizer(object, CDNFinalizer)
}

func HasGroupAnnotation(o client.Object) bool {
	return len(o.GetAnnotations()[CDNGroupAnnotation]) > 0
}

func HasLoadBalancer(o client.Object) bool {
	ingv1beta1, ok := o.(*v1beta1.Ingress)
	if ok {
		return len(ingv1beta1.Status.LoadBalancer.Ingress) > 0 && len(ingv1beta1.Status.LoadBalancer.Ingress[0].Hostname) > 0
	}

	ingv1, ok := o.(*v1.Ingress)
	if !ok {
		return false
	}

	return len(ingv1.Status.LoadBalancer.Ingress) > 0 && len(ingv1.Status.LoadBalancer.Ingress[0].Hostname) > 0
}
