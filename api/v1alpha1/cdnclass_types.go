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

// CDNClassSpec defines the desired state of CDNClass
type CDNClassSpec struct {
	// CertificateArn is deprecated, certs are now automatically discovered and this field is ignored
	// +optional
	CertificateArn string `json:"certificateArn"`
	// HostedZoneID represents a valid hosted zone ID for a domain name
	// +kubebuilder:validation:Required
	HostedZoneID string `json:"hostedZoneID"`
	// CreateAlias determine if should create an DNS alias for distribution
	// +kubebuilder:validation:Required
	CreateAlias bool `json:"createAlias"`
	// TXTOwnerValue is the value to be used when creating ownership TXT records for aliases
	// +kubebuilder:validation:Required
	TXTOwnerValue string `json:"txtOwnerValue"`
}

// CDNClassStatus defines the observed state of CDNClass
type CDNClassStatus struct {
	// CDNClass resource condition
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// CDNClass is the Schema for the cdnclasses API
type CDNClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDNClassSpec   `json:"spec,omitempty"`
	Status CDNClassStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CDNClassList contains a list of CDNClass
type CDNClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDNClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDNClass{}, &CDNClassList{})
}
