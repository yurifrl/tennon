package domain

import (
	"encoding/json"
	"time"
)

// Execution represents a paused tool call awaiting approval. State transitions:
// pending → approved → completed (on success) or failed (on error)
// pending → denied (approval rejected)
type Execution struct {
	Meta   ObjectMeta     `json:"metadata"`
	Spec   ExecutionSpec  `json:"spec"`
	Status ExecutionStatus `json:"status,omitempty"`
}

// ExecutionSpec is the immutable desired state of an Execution.
type ExecutionSpec struct {
	Subject   string         `json:"subject"`            // authenticated subject who made the call
	Connector string         `json:"connector"`          // target connector name
	Tool      string         `json:"tool"`               // tool name (exposed, with prefix)
	Arguments map[string]any `json:"arguments,omitempty"` // tool call arguments
}

// ExecutionStatus is the mutable runtime state of an Execution.
type ExecutionStatus struct {
	State      ExecutionState `json:"state"`
	Created    time.Time      `json:"created,omitempty"`
	Approved   time.Time      `json:"approved,omitempty"`
	Completed  time.Time      `json:"completed,omitempty"`
	Denied     time.Time      `json:"denied,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`     // upstream tool result (on completion)
	Error      string          `json:"error,omitempty"`      // error message (on failure)
}

// ExecutionState is the lifecycle state of an Execution.
type ExecutionState string

const (
	ExecutionPending   ExecutionState = "pending"
	ExecutionApproved  ExecutionState = "approved"
	ExecutionCompleted ExecutionState = "completed"
	ExecutionDenied    ExecutionState = "denied"
	ExecutionFailed    ExecutionState = "failed"
)

// ToResource marshals an Execution into a store Resource.
func (e *Execution) ToResource() (*Resource, error) {
	spec, err := json.Marshal(e.Spec)
	if err != nil {
		return nil, err
	}
	status, err := json.Marshal(e.Status)
	if err != nil {
		return nil, err
	}
	return &Resource{Kind: KindExecution, Meta: e.Meta, Spec: spec, Status: status}, nil
}

// ExecutionFromResource unmarshals a store Resource into an Execution.
func ExecutionFromResource(r *Resource) (*Execution, error) {
	e := &Execution{Meta: r.Meta}
	if len(r.Spec) > 0 {
		if err := json.Unmarshal(r.Spec, &e.Spec); err != nil {
			return nil, err
		}
	}
	if len(r.Status) > 0 {
		if err := json.Unmarshal(r.Status, &e.Status); err != nil {
			return nil, err
		}
	}
	return e, nil
}
