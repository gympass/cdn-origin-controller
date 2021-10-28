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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CDNStatusSpec defines the desired state of CDNStatus
type CDNStatusSpec struct {
}

// IngressRefs ingresses list
type IngressRefs []IngressRef

// IngressRef ingress reference
type IngressRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	InSync    bool   `json:"in-sync,omitempty"`
}

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

	//Spec   CDNStatusSpec   `json:"spec,omitempty"`
	Status CDNStatusStatus `json:"status,omitempty"`
}

type namespacedName interface {
	GetNamespace() string
	GetName() string
}

// SetRef set IngressRef to the status based on Ingress obj
func (c *CDNStatus) SetRef(inSync bool, obj namespacedName) {
	ingresses := c.Status.Ingresses
	for i, ingRef := range ingresses {
		if ingRef.Name == obj.GetName() && ingRef.Namespace == obj.GetNamespace() {
			ingresses[i].InSync = inSync
			return
		}
	}

	ingRef := IngressRef{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		InSync:    inSync,
	}
	c.Status.Ingresses = append(ingresses, ingRef)
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
