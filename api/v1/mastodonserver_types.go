package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MastodonServerSpec defines the desired state of MastodonServer
type MastodonServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MastodonServer. Edit mastodon_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MastodonServerStatus defines the observed state of MastodonServer
type MastodonServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MastodonServer is the Schema for the mastodons API
type MastodonServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MastodonServerSpec   `json:"spec,omitempty"`
	Status MastodonServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MastodonServerList contains a list of MastodonServer
type MastodonServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MastodonServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MastodonServer{}, &MastodonServerList{})
}
