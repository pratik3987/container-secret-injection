package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// VaultInjectorSpec defines the desired state.
type VaultInjectorSpec struct {
	// ServiceName is the service that fronts the webhook.
	ServiceName string `json:"serviceName"`
	// CASecret is the name of the secret containing ca.crt for webhook.
	CASecret string `json:"caSecret"`
	// Namespace where the webhook service lives.
	ServiceNamespace string `json:"serviceNamespace"`
}

// VaultInjectorStatus defines observed state.
type VaultInjectorStatus struct {
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// VaultInjector is the CRD type for the webhook injector.
type VaultInjector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultInjectorSpec   `json:"spec,omitempty"`
	Status VaultInjectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// VaultInjectorList contains a list of VaultInjector resources.
type VaultInjectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultInjector `json:"items"`
}

func (in *VaultInjector) DeepCopyInto(out *VaultInjector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = in.Spec
	out.Status = in.Status
}

func (in *VaultInjector) DeepCopy() *VaultInjector {
	if in == nil {
		return nil
	}
	out := new(VaultInjector)
	in.DeepCopyInto(out)
	return out
}

func (in *VaultInjector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *VaultInjectorList) DeepCopyInto(out *VaultInjectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]VaultInjector, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *VaultInjectorList) DeepCopy() *VaultInjectorList {
	if in == nil {
		return nil
	}
	out := new(VaultInjectorList)
	in.DeepCopyInto(out)
	return out
}

func (in *VaultInjectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
