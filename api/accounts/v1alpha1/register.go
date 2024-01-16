package v1alpha1

import "k8s.io/apimachinery/pkg/runtime/schema"

// +kubebuilder:rbac:groups=core,resources=events,verbs=create

var SchemeGroupVersion = GroupVersion

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
