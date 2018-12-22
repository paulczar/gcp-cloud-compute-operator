package v1

import (
	"github.com/jinzhu/copier"
	gce "google.golang.org/api/iam/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceAccountStatus defines the observed state of ServiceAccount
type ServiceAccountStatus struct {
	ProjectId      string `json:"projectId,omitempty"`
	UniqueId       string `json:"uniqueId,omitempty"`
	Oauth2ClientId string `json:"oauth2ClientId,omitempty"`
	Email          string `json:"email,omitempty"`
	Status         string `json:"status,omitempty"`
	Name           string `json:"name,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccount is the Schema for the serviceaccounts API
// +k8s:openapi-gen=true
type ServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *gce.ServiceAccount  `json:"spec,omitempty"`
	Status ServiceAccountStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccountList contains a list of ServiceAccount
type ServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceAccount{}, &ServiceAccountList{})
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceAccount) DeepCopyInto(out *ServiceAccount) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Status.DeepCopyInto(&out.Status)
	copier.Copy(&in.Spec, &out.Spec)
	return
}
