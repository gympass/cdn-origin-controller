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

	"github.com/Gympass/cdn-origin-controller/internal/strhelper"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DNSStatus provides status regarding the creation of DNS records for aliases
type DNSStatus struct {
	Records []string `json:"records,omitempty"`
	Synced  bool     `json:"synced,omitempty"`
}

// CDNStatusStatus defines the observed state of CDNStatus
type CDNStatusStatus struct {
	ID        string      `json:"id,omitempty"`
	ARN       string      `json:"arn,omitempty"`
	Ingresses IngressRefs `json:"ingresses,omitempty"`
	Aliases   []string    `json:"aliases,omitempty"`
	Address   string      `json:"address,omitempty"`
	// +optional
	// +nullable
	DNS *DNSStatus `json:"dns,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Aliases",type=string,JSONPath=`.status.aliases`
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.address`

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

// SetAliases sets the CDN's aliases
func (c *CDNStatus) SetAliases(aliases []string) {
	c.Status.Aliases = aliases
}

// UpsertDNSRecords inserts the given records at the DNS status section if they're not present already
func (c *CDNStatus) UpsertDNSRecords(records []string) {
	if len(records) == 0 {
		return
	}

	if c.Status.DNS == nil {
		c.Status.DNS = &DNSStatus{Records: records}
		return
	}

	for _, r := range records {
		if !strhelper.Contains(c.Status.DNS.Records, r) {
			c.Status.DNS.Records = append(c.Status.DNS.Records, r)
		}
	}
}

// RemoveDNSRecords deletes the given records from the DNS status section
func (c *CDNStatus) RemoveDNSRecords(records []string) {
	for _, it := range records {
		predicate := func(i string) bool {
			return i != it
		}
		c.Status.DNS.Records = strhelper.Filter(c.Status.DNS.Records, predicate)
	}
}

// SetIngressRef set IngressRef to the status based on Ingress obj
func (c *CDNStatus) SetIngressRef(inSync bool, ing namespacedName) {
	if c.Status.Ingresses == nil {
		c.Status.Ingresses = make(IngressRefs)
	}

	ref := NewIngressRef(ing.GetNamespace(), ing.GetName())

	status := syncedIngressStatus
	if !inSync {
		status = failedIngressStatus
	}

	c.Status.Ingresses[ref] = status
}

// RemoveIngressRef ensures the given Ingress is not referenced
func (c *CDNStatus) RemoveIngressRef(ing namespacedName) {
	if c.Status.Ingresses == nil {
		return
	}
	ref := NewIngressRef(ing.GetNamespace(), ing.GetName())
	delete(c.Status.Ingresses, ref)
}

// HasIngressRef returns whether the given Ingress is part of the refs stored by CDNStatus
func (c *CDNStatus) HasIngressRef(ing namespacedName) bool {
	if c.Status.Ingresses == nil {
		return false
	}
	ref := NewIngressRef(ing.GetNamespace(), ing.GetName())
	_, ok := c.Status.Ingresses[ref]
	return ok
}

// GetIngressKeys returns keys to manipulate all IngressRef stored in the CDNStatus
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
