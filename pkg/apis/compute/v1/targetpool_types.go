package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TargetPoolStatus defines the observed state of TargetPool
type TargetPoolStatus struct {
	Status            string `json:status,omitempty`
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
	Id                uint64 `json:"id,omitempty,string"`
	Region            string `json:"region,omitempty"`
	SelfLink          string `json:"selfLink,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetPool is the Schema for the targetpools API
// +k8s:openapi-gen=true
type TargetPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   map[string]interface{} `json:"spec,omitempty"`
	Status TargetPoolStatus       `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TargetPoolList contains a list of TargetPool
type TargetPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TargetPool `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TargetPool{}, &TargetPoolList{})
}
