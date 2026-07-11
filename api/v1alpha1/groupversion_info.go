package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "vault.prtk.com"
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion, &VaultInjector{}, &VaultInjectorList{})
	metav1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

func AddToScheme(s *runtime.Scheme) error { return SchemeBuilder.AddToScheme(s) }
