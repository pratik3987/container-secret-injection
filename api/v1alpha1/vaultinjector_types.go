package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VaultInjectorSpec defines the desired state
type VaultInjectorSpec struct {
	// ServiceName is the service that fronts the webhook
	ServiceName string `json:"serviceName"`
	// CASecret is the name of the secret containing ca.crt for webhook
	CASecret string `json:"caSecret"`
	// Namespace where the webhook service lives
	ServiceNamespace string `json:"serviceNamespace"`
}

// VaultInjectorStatus defines observed state
type VaultInjectorStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type VaultInjector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultInjectorSpec   `json:"spec,omitempty"`
	Status VaultInjectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type VaultInjectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultInjector `json:"items"`
}
