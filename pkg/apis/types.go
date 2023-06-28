package apis

// +k8s:deepcopy-gen=false
type ConditionManagerAccessor interface {
	GetConditionManager() ConditionManager
}
