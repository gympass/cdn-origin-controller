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
	v1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/Gympass/cdn-origin-controller/internal/config"
	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

// CDNClassAnnotationValue returns the CDN class found within an Ingress' annotations
func CDNClassAnnotationValue(object client.Object) string {
	return object.GetAnnotations()[CDNClassAnnotation]
}

// CDNClassMatches returns whether the candidate class matches the one being managed by this controller
func CDNClassMatches(candidate string) bool {
	return candidate == config.CDNClass()
}

// HasFinalizer returns whether a given Ingress has a finalizer managed by this controller
func HasFinalizer(object client.Object) bool {
	return strhelper.Contains(object.GetFinalizers(), CDNFinalizer)
}

// AddFinalizer adds the finalizer managed by this controller to a given Ingress.
// It does not make any calls to the API server.
func AddFinalizer(object client.Object) {
	controllerutil.AddFinalizer(object, CDNFinalizer)
}

// RemoveFinalizer removes the finalizer managed by this controller from a given Ingress.
// It does not make any calls to the API server.
func RemoveFinalizer(object client.Object) {
	controllerutil.RemoveFinalizer(object, CDNFinalizer)
}

// HasGroupAnnotation returns whether the given Ingress has the CDN group annotation
func HasGroupAnnotation(o client.Object) bool {
	return len(o.GetAnnotations()[CDNGroupAnnotation]) > 0
}

// HasLoadBalancer returns whether the given Ingress has been provisioned
func HasLoadBalancer(o client.Object) bool {
	ingv1, ok := o.(*v1.Ingress)
	if !ok {
		return false
	}

	return len(ingv1.Status.LoadBalancer.Ingress) > 0 && len(ingv1.Status.LoadBalancer.Ingress[0].Hostname) > 0
}

// IsBeingRemovedFromDesiredState return whether the Ingress is being removed or if it no longer belongs to a group
func IsBeingRemovedFromDesiredState(obj client.Object) bool {
	return obj.GetDeletionTimestamp() != nil || (!HasGroupAnnotation(obj) && HasFinalizer(obj))
}
