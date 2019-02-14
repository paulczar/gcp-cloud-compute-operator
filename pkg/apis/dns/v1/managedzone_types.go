package v1

import (
	"github.com/jinzhu/copier"
	gce "google.golang.org/api/dns/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ManagedZoneStatus defines the observed state of ManagedZone
type ManagedZoneStatus struct {
	CreationTime string   `json:"creationTime,omitempty"`
	Status       string   `json:"status,omitempty"`
	Id           uint64   `json:"id,omitempty,string"`
	NameServers  []string `json:"nameServers,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagedZone is the Schema for the managedzones API
// +k8s:openapi-gen=true
type ManagedZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *gce.ManagedZone  `json:"spec,omitempty"`
	Status ManagedZoneStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagedZoneList contains a list of ManagedZone
type ManagedZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedZone{}, &ManagedZoneList{})
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedZone) DeepCopyInto(out *ManagedZone) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Status.DeepCopyInto(&out.Status)
	copier.Copy(&in.Spec, &out.Spec)
	return
}
