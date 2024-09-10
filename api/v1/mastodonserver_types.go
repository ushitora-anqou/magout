package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type PeriodicRestartSpec struct {
	Schedule string  `json:"schedule"`
	TimeZone *string `json:"timeZone,omitempty"`
}

type MastodonServerSidekiqSpec struct {
	Image           string                      `json:"image"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Labels          map[string]string           `json:"labels,omitempty"`
	Annotations     map[string]string           `json:"annotations,omitempty"`
	PodAnnotations  map[string]string           `json:"podAnnotations,omitempty"`
	PeriodicRestart *PeriodicRestartSpec        `json:"periodicRestart,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`
}

type MastodonServerStreamingSpec struct {
	Image           string                      `json:"image"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Labels          map[string]string           `json:"labels,omitempty"`
	Annotations     map[string]string           `json:"annotations,omitempty"`
	PodAnnotations  map[string]string           `json:"podAnnotations,omitempty"`
	PeriodicRestart *PeriodicRestartSpec        `json:"periodicRestart,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`
}

type MastodonServerWebSpec struct {
	Image           string                      `json:"image"`
	EnvFrom         []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	Labels          map[string]string           `json:"labels,omitempty"`
	Annotations     map[string]string           `json:"annotations,omitempty"`
	PodAnnotations  map[string]string           `json:"podAnnotations,omitempty"`
	PeriodicRestart *PeriodicRestartSpec        `json:"periodicRestart,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`
}

// MastodonServerSpec defines the desired state of MastodonServer.
type MastodonServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Web       MastodonServerWebSpec       `json:"web"`
	Streaming MastodonServerStreamingSpec `json:"streaming"`
	Sidekiq   MastodonServerSidekiqSpec   `json:"sidekiq"`
}

type MastodonServerMigratingStatus struct {
	Web       string `json:"web"`
	Sidekiq   string `json:"sidekiq"`
	Streaming string `json:"streaming"`
}

// MastodonServerStatus defines the observed state of MastodonServer.
type MastodonServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Migrating *MastodonServerMigratingStatus `json:"migrating,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MastodonServer is the Schema for the mastodons API.
type MastodonServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MastodonServerSpec   `json:"spec,omitempty"`
	Status MastodonServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MastodonServerList contains a list of MastodonServer.
type MastodonServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MastodonServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MastodonServer{}, &MastodonServerList{})
}
