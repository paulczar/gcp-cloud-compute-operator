package v1

import (
	"github.com/jinzhu/copier"
	gce "google.golang.org/api/compute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddressStatus defines the observed state of Address
type AddressStatus struct {
	Status    string `json:"status,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
	SelfLink  string `json:"selfLink,omitempty"`
	Region    string `json:"region,omitempty"`
}

// +k8s:deepcopy-gen=false

// AddressSpec is an interface
//type AddressSpec struct {
//	*gceCompute.Address
//}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Address is the Schema for the addresses API
// +k8s:openapi-gen=true
type Address struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *gce.Address  `json:"spec,omitempty"`
	Status AddressStatus `json:"status,omitempty"`
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

// DeepCopyInto is a deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Address) DeepCopyInto(out *Address) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Status.DeepCopyInto(&out.Status)
	copier.Copy(&in.Spec, &out.Spec)
	return
}
