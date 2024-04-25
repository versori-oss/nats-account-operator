package resources

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type SecretOption func(secret *v1.Secret) error

func Immutable() SecretOption {
	return func(secret *v1.Secret) error {
		secret.Immutable = ptr.To(true)

		return nil
	}
}
