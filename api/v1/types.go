package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	Schedule string         `json:"schedule"`
	Tasks    []Workflowtask `json:"tasks"`
}

type Workflowtask struct {
	Name string `json:"name"`
	//Type    string   `json:"type"`
	Command struct {
		Inline struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"inline"`

		Script string `json:"script"`
	} `json:"command"`
	//Args []string `json:"args"`
}

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	Runs []Workflowruns `json:"runs"`
}

type Workflowruns struct {
	ID    int          `json:"id"`
	Phase string       `json:"phase"`
	Tasks []TaskStatus `json:"tasks"`
}

type TaskStatus struct {
	Name string `json:"name"`
	//Command string   `json:"command"`
	//Args    []string `json:"args"`
	Status string `json:"status"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   WorkflowSpec   `json:"spec"`
	Status WorkflowStatus `json:"status"`
}

// WorkflowList contains a list of Workflow
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
