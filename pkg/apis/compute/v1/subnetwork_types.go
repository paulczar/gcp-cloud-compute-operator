package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SubnetworkStatus defines the observed state of Subnetwork
type SubnetworkStatus struct {
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
	Id                uint64 `json:"id,omitempty,string"`
	SelfLink          string `json:"selfLink,omitempty"`
	Status            string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Subnetwork is the Schema for the subnetworks API
// +k8s:openapi-gen=true
type Subnetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   map[string]interface{} `json:"spec,omitempty"`
	Status SubnetworkStatus       `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubnetworkList contains a list of Subnetwork
type SubnetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnetwork{}, &SubnetworkList{})
}
