package controllers

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type SecretOpt func(*v1.Secret)

// WithData sets the data of the secret.
func WithData(data map[string][]byte) SecretOpt {
	return func(s *v1.Secret) {
		s.Data = data
	}
}

// WithStringData sets the string data of the secret.
func WithStringData(data map[string]string) SecretOpt {
	return func(s *v1.Secret) {
		s.StringData = data
	}
}

// WithImmutable sets the immutable field of the secret.
func WithImmutable(immutable bool) SecretOpt {
	return func(s *v1.Secret) {
		s.Immutable = ptr.To(immutable)
	}
}

// WithLabels sets the labels of the ObjectMeta in the secret.
func WithLabels(labels map[string]string) SecretOpt {
	return func(s *v1.Secret) {
		s.ObjectMeta.Labels = labels
	}
}

// WithAnnotations sets the annotations of the ObjectMeta in the secret.
func WithAnnotations(annotations map[string]string) SecretOpt {
	return func(s *v1.Secret) {
		s.ObjectMeta.Annotations = annotations
	}
}

// NewSecret creates a new secret with the given name and namespace. Other options can be passed in to set desired fields.
func NewSecret(name, namespace string, opts ...SecretOpt) v1.Secret {
	s := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	for _, opt := range opts {
		opt(&s)
	}

	return s
}
