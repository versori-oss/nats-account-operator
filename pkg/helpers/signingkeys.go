package helpers

import (
	"k8s.io/apimachinery/pkg/types"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

// NextSigningKeys compares the current SigningKeys assigned to a resource to a list of SigningKeys which currently
// exist on the cluster and returns the next list of SigningKeys to be assigned to the resource.
//
// This function attempts to preserve the order of SigningKeys on the resource, so that no-op updates do not trigger an
// update on the status. New SigningKeys are appended to the end of the list, and SigningKeys which are no longer active
// are removed from the list. Any changes to existing SigningKeys are kept at the same index (however removals may cause
// indices to shift).
func NextSigningKeys(ownerUID types.UID, current []v1alpha1.SigningKeyEmbeddedStatus, next *v1alpha1.SigningKeyList) []v1alpha1.SigningKeyEmbeddedStatus {
	existingSKs := make(map[string]v1alpha1.SigningKeyEmbeddedStatus)
	for _, sk := range current {
		existingSKs[sk.Name] = sk
	}

	// create a map of SigningKeys by name for easier lookup, removing any we find which already exist on the status
	nextSKsByName := make(map[string]v1alpha1.SigningKeyEmbeddedStatus)
	for _, sk := range next.Items {
		// this SigningKey is not ready or is owned by another account
		if !sk.Status.IsReady() || sk.Status.OwnerRef.UID != ownerUID {
			continue
		}

		nextSKsByName[sk.GetName()] = v1alpha1.SigningKeyEmbeddedStatus{
			Name:    sk.GetName(),
			KeyPair: *sk.Status.KeyPair,
		}
	}

	nextSKs := make([]v1alpha1.SigningKeyEmbeddedStatus, 0, len(nextSKsByName))

	for _, existing := range current {
		next, ok := nextSKsByName[existing.Name]

		if !ok {
			// this SigningKey no longer active on this Account, so we don't need to add to nextSKs
			continue
		}

		// add the SigningKey to the nextSKs slice
		nextSKs = append(nextSKs, next)

		// remove the SigningKey from nextSKsByName so that we can check for any SigningKeys which need to be appended
		// at the end
		delete(nextSKsByName, existing.Name)
	}

	// whatever remains in nextSKsByName are new SigningKeys we weren't previously aware of
	for _, next := range nextSKsByName {
		nextSKs = append(nextSKs, next)
	}

	return nextSKs
}
