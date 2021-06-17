package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TenantTemplate is the Schema for the tenant templates API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=tenanttemplates,scope=Namespaced
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type TenantTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TenantSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// TenantTemplateList contains a list of TenantTemplate
type TenantTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantTemplate{}, &TenantTemplateList{})
}
