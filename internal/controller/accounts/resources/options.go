package resources

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SecretOption func(secret *v1.Secret) error

func Immutable() SecretOption {
	return func(secret *v1.Secret) error {
		secret.Immutable = ptr.To(true)

		return nil
	}
}

// WithDeletionPrevention adds a finalizer to the secret which will never be removed by this
// controller. This is useful for Operator and Account seed secrets which can be very destructive
// if deleted accidentally. Users will still need to recreate the seed if they trigger a deletion
// since the deletionTimestamp will be set, but the finalizer will prevent Kubernetes from
// garbage collecting, giving them time to copy the secret in preparation for recreating.
func WithDeletionPrevention() SecretOption {
	return func(secret *v1.Secret) error {
		controllerutil.AddFinalizer(secret, "accounts.versori.io/deletion-prevention")

		return nil
	}
}
