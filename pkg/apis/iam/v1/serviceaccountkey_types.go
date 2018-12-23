package v1

import (
	"github.com/jinzhu/copier"
	gce "google.golang.org/api/iam/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountKeyStatus defines the observed state of ServiceAccountKey
type ServiceAccountKeyStatus struct {
	Status string `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccountKey is the Schema for the serviceaccountkeys API
// +k8s:openapi-gen=true
type ServiceAccountKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *gce.ServiceAccountKey  `json:"spec,omitempty"`
	Status ServiceAccountKeyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccountKeyList contains a list of ServiceAccountKey
type ServiceAccountKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccountKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceAccountKey{}, &ServiceAccountKeyList{})
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceAccountKey) DeepCopyInto(out *ServiceAccountKey) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Status.DeepCopyInto(&out.Status)
	copier.Copy(&in.Spec, &out.Spec)
	return
}
