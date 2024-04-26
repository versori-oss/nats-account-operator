package helpers

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MatchNamespaceSelector(parent client.Object, childNamespace *v1.Namespace, selector *metav1.LabelSelector) (bool, error) {
	if selector == nil {
		// If no selector is provided, the namespace is allowed if both namespaces are the same.
		return parent.GetNamespace() == childNamespace.Name, nil
	}

	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false, fmt.Errorf("failed to compile label selector: %w", err)
	}

	return s.Matches(labels.Set(childNamespace.GetLabels())), nil
}
