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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImagePolicySpec defines the desired state of ImagePolicy
type ImagePolicySpec struct {
	Validator string `json:"validator"`
	// Pattern contains reference to the image that has to be validated by this policy
	Pattern string `json:"pattern"`

	//+kubebuilder:default:value=false
	// Strict defines, if the policy should warn, or prohibit scheduling
	Strict bool `json:"strict"`
}

// ImagePolicyStatus defines the observed state of ImagePolicy
type ImagePolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

//+kubebuilder:printcolumn:name=validator,type=string,JSONPath=.spec.validator
//+kubebuilder:printcolumn:name=pattern,type=string,JSONPath=.spec.pattern
//+kubebuilder:printcolumn:name=strict,type=boolean,JSONPath=.spec.strict

// ImagePolicy is the Schema for the imagepolicies API
type ImagePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ImagePolicySpec `json:"spec,omitempty"`
	//Status ImagePolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImagePolicyList contains a list of ImagePolicy
type ImagePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImagePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImagePolicy{}, &ImagePolicyList{})
}
