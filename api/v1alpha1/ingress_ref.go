package v1alpha1

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

const separator = "/"

// IngressRef represents a reference to an Ingress
type IngressRef string

// NewIngressRef creates a new IngressRef for given name and namespace
func NewIngressRef(namespace, name string) IngressRef {
	return IngressRef(namespace + separator + name)
}

// GetNamespace returns the IngressRef namespace
func (i IngressRef) GetNamespace() string {
	parts := strings.Split(string(i), separator)
	return parts[0]
}

// GetName returns the IngressRef name
func (i IngressRef) GetName() string {
	parts := strings.Split(string(i), separator)
	return parts[1]
}

// ToNamespacedName creates a types.Namespaced from an IngressRef
func (i IngressRef) ToNamespacedName() types.NamespacedName {
	parts := strings.Split(string(i), separator)
	return types.NamespacedName{
		Namespace: parts[0],
		Name:      parts[1],
	}
}

// IngressRefs ingresses map
type IngressRefs map[IngressRef]string
