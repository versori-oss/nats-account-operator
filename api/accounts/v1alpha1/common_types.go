package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type NatsSecretType string

const (
	NatsSecretTypeSeed NatsSecretType = "seed"
	NatsSecretTypeJWT  NatsSecretType = "jwt"
	NatsSecretTypeSKey NatsSecretType = "skey"
)

const (
	NatsSecretJWTKey       = "nats.jwt"
	NatsSecretCredsKey     = "nats.creds"
	NatsCAKey              = "ca.crt"
	NatsSecretSeedKey      = "seed.nk"
	NatsSecretPublicKeyKey = "public.nk"
)

// InferredObjectReference is an object reference without the APIVersion and Kind fields. The APIVersion and Kind
// are inferred based on where the reference is used.
type InferredObjectReference struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

type TypedObjectReference struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace,omitempty"`
	UID        types.UID `json:"uid,omitempty"`
}

func (r *TypedObjectReference) GetGroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(r.APIVersion, r.Kind)
}

type Identity struct {
	ID    string `json:"id"`
	Proof string `json:"proof"`
}

type SigningKeyEmbeddedStatus struct {
	Name    string  `json:"name"`
	KeyPair KeyPair `json:"keyPair,omitempty"`
}

// IssuerReference provides the means to look up a signing key for generating an Account or User.
type IssuerReference struct {
	Ref TypedObjectReference `json:"ref"`
}

// SigningKeyOwnerReference provides the means to reference the owning object for a signing key. This should be one of
// Operator or Account.
type SigningKeyOwnerReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

type KeyPairReference struct {
	InferredObjectReference `json:",inline"`
	PublicKey string `json:"publicKey"`
}
