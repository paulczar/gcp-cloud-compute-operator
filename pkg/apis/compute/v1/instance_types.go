package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	CpuPlatform       string `json:"cpuPlatform,omitempty"`
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
	Id                uint64 `json:"id,omitempty,string"`
	SelfLink          string `json:"selfLink,omitempty"`
	StartRestricted   bool   `json:"startRestricted,omitempty"`
	Status            string `json:"status,omitempty"`
	StatusMessage     string `json:"statusMessage,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Instance is the Schema for the instances API
// +k8s:openapi-gen=true
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   map[string]interface{} `json:"spec,omitempty"`
	Status InstanceStatus         `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceList contains a list of Instance
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
