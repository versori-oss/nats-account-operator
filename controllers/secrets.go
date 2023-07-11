package controllers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/pointer"
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
		s.Immutable = pointer.Bool(immutable)
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

// createOrUpdateSecret creates or updates the given secret. Pass update=true to update an existing secret.
// Passing false will create it. This function is to tidy up code in the controllers.
func createOrUpdateSecret(ctx context.Context, CV1Interface corev1.CoreV1Interface, namespace string, secret *v1.Secret, update bool) (*v1.Secret, error) {
	var err error
	var returnedSecret *v1.Secret

	if update {
		returnedSecret, err = CV1Interface.Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	} else {
		returnedSecret, err = CV1Interface.Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	}
	return returnedSecret, err
}
