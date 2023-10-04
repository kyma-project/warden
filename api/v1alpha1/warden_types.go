package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen=true

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Warden is the Schema for the serverlesses API
type Warden struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WardenSpec   `json:"spec,omitempty"`
	Status WardenStatus `json:"status,omitempty"`
}

type WardenSpec struct {
	Ready string `json:"ready,omitempty"`
}

type WardenStatus struct{}

//+kubebuilder:object:root=true

// WardenList contains a list of Serverless
type WardenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Warden `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Warden{}, &WardenList{})
}
