package v1alpha1

import "github.com/versori-oss/nats-account-operator/pkg/apis"

type Status struct {
	// Conditions the latest available observations of a resource's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions apis.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

var _ apis.ConditionsAccessor = (*Status)(nil)

func (s *Status) GetConditions() apis.Conditions {
	return s.Conditions
}

func (s *Status) SetConditions(conditions apis.Conditions) {
	s.Conditions = conditions
}

// +k8s:deepcopy-gen=false

// StatusAccessor provides a way to access our standard Status subresource which contains Conditions.
type StatusAccessor interface {
	GetStatus() *Status
}

// +k8s:deepcopy-gen=false

// ConditionSetAccessor provides a way to access a resource's ConditionSet.
type ConditionSetAccessor interface {
	GetConditionSet() apis.ConditionSet
}
