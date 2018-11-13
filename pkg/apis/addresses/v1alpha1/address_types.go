package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AddressSpec defines the desired state of Address
// We use map[string]interface{} instead of this
//type AddressSpec struct {}

// AddressStatus defines the observed state of Address
type AddressStatus struct {
	Status    string `json:"status,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	SelfLink  string `json:"selfLink,omitempty"`
	Region    string `json:"region,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Address is the Schema for the addresses API
// +k8s:openapi-gen=true
type Address struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   map[string]interface{} `json:"spec,omitempty"`
	Status AddressStatus          `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddressList contains a list of Address
type AddressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Address `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Address{}, &AddressList{})
}
