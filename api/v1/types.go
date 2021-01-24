package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	Schedule string         `json:"schedule,omitempty"`
	Tasks    []workflowtask `json:"tasks,omitempty"`
}

type workflowtask struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
	Data string `json:"data,omitempty"`
}

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
}

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// WorkflowList contains a list of Workflow
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
