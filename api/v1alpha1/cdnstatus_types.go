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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CDNStatusStatus defines the observed state of CDNStatus
type CDNStatusStatus struct {
	ID        string      `json:"id,omitempty"`
	ARN       string      `json:"arn,omitempty"`
	Ingresses IngressRefs `json:"ingresses,omitempty"`
	Aliases   []string    `json:"aliases,omitempty"`
	Address   string      `json:"address,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// CDNStatus is the Schema for the cdnstatuses API
type CDNStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status CDNStatusStatus `json:"status,omitempty"`
}

const (
	failedIngressStatus = "Failed"
	syncedIngressStatus = "Synced"
)

type namespacedName interface {
	GetNamespace() string
	GetName() string
}

// SetIngressRef set IngressRef to the status based on Ingress obj
func (c *CDNStatus) SetIngressRef(inSync bool, obj namespacedName) {
	if c.Status.Ingresses == nil {
		c.Status.Ingresses = make(IngressRefs)
	}

	ref := NewIngressRef(obj.GetNamespace(), obj.GetName())

	status := syncedIngressStatus
	if !inSync {
		status = failedIngressStatus
	}

	c.Status.Ingresses[ref] = status
}

func (c *CDNStatus) GetIngressKeys() []client.ObjectKey {
	var keys []types.NamespacedName
	for ing := range c.Status.Ingresses {
		keys = append(keys, ing.ToNamespacedName())
	}
	return keys
}

//+kubebuilder:object:root=true

// CDNStatusList contains a list of CDNStatus
type CDNStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDNStatus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDNStatus{}, &CDNStatusList{})
}
