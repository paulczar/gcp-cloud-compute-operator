package v1

import (
	"github.com/jinzhu/copier"
	gce "google.golang.org/api/compute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ForwardingRuleStatus defines the observed state of ForwardingRule
type ForwardingRuleStatus struct {
	Status            string `json:"status,omitempty"`
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
	Id                uint64 `json:"id,omitempty,string"`
	SelfLink          string `json:"selfLink,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ForwardingRule is the Schema for the forwardingrules API
// +k8s:openapi-gen=true
type ForwardingRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *gce.ForwardingRule  `json:"spec,omitempty"`
	Status ForwardingRuleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ForwardingRuleList contains a list of ForwardingRule
type ForwardingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ForwardingRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ForwardingRule{}, &ForwardingRuleList{})
}

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRule) DeepCopyInto(out *ForwardingRule) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Status.DeepCopyInto(&out.Status)
	copier.Copy(&in.Spec, &out.Spec)
	return
}
