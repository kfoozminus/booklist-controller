package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const NamespaceDefault string = "default"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Jackpot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JackpotSpec   `json:"spec,omitempty"`
	Status JackpotStatus `json:"status"`
}

type JackpotSpec struct {
	Image    string `json:"image,omitempty"`
	Replicas *int32 `json:"replicas,omitempty"`
}

type JackpotStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type JackpotList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Items []Jackpot `json:"items,omitempty"`
}
