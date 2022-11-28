/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/kyma-project/warden/pkg/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ValidatorSpec defines the desired state of Validator
type ValidatorSpec struct {
	// Type contains the type of the defined validator
	//+kubebuilder:validation:Enum=notary;allow;deny
	Type string `json:"type"`
	// NotaryConfig contains specific information for a validator that uses Notary as it's backend
	NotaryConfig validate.NotaryConfig `json:"notaryConfig"`
}

// ValidatorStatus defines the observed state of Validator
type ValidatorStatus struct {
	// State contains current state of validator availability
	// Allowed states are "available" and "unavailable"
	State string `json:"state"`
}

// +kubebuilder:subresource:status
// +kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// Validator is the Schema for the validators API
// +kubebuilder:printcolumn:name=state,type=string,JSONPath=.status.state
// +kubebuilder:printcolumn:name=age,type=date,JSONPath=.metadata.creationTimestamp
type Validator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ValidatorSpec   `json:"spec,omitempty"`
	Status ValidatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ValidatorList contains a list of Validator
type ValidatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Validator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Validator{}, &ValidatorList{})
}
