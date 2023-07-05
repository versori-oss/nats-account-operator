package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// KeyPairableConditionSeedSecretReady is a condition type which should apply to all KeyPairable resources, and denotes
	// whether the SeedSecret is ready and the .status.keyPair field is populated.
	KeyPairableConditionSeedSecretReady = "SeedSecretReady"
)

// KeyPair is the reference to the KeyPair that will be used to sign JWTs for Accounts and Users.
type KeyPair struct {
	PublicKey      string `json:"publicKey"`
	SeedSecretName string `json:"seedSecretName"`
}

// +k8s:deepcopy-gen=false

// KeyPairable is an interface which should be implemented by all resources which have a KeyPair to sign JWTs.
type KeyPairable interface {
	metav1.Object
	schema.ObjectKind

	StatusAccessor
	ConditionSetAccessor

	GetKeyPair() *KeyPair
}
