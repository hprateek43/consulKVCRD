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

// ConsulKVSpec defines the desired state of ConsulKV
type ConsulKVSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ConsulKV. Edit consulkv_types.go to remove/update
	//Keys map[string]string           `json:"keys,omitempty"`
	Keys string `json:"keys,omitempty"`
}

// ConsulKVStatus defines the observed state of ConsulKV
type ConsulKVStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ConsulServer string   `json:"consulServer,omitempty"`
	Keys         []string `json:"keys,omitempty"`
	LastSynced   string   `json:"lastSynced,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ConsulKV is the Schema for the consulkvs API
type ConsulKV struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConsulKVSpec   `json:"spec,omitempty"`
	Status ConsulKVStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConsulKVList contains a list of ConsulKV
type ConsulKVList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConsulKV `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConsulKV{}, &ConsulKVList{})
}
