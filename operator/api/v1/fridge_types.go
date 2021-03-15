package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FridgeSpec defines the desired state of Fridge
type FridgeSpec struct {
	// MQTT topic of the fridge to monitor.
	// +kubebuilder:validation:Required
	Topic string `json:"topic"`
}

// FridgeStatus defines the observed state of Fridge
type FridgeStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Fridge is the Schema for the fridges API
// +kubebuilder:printcolumn:name="Topic",type=string,JSONPath=`.spec.topic`
type Fridge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FridgeSpec   `json:"spec,omitempty"`
	Status FridgeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FridgeList contains a list of Fridge
type FridgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fridge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Fridge{}, &FridgeList{})
}
