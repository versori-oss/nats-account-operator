package v1alpha1

type KeyPairAccessor interface {
	GetKeyPair() *KeyPair
}

type KeyPair struct {
	PublicKey      string `json:"publicKey"`
	SeedSecretName string `json:"seedSecretName"`
}
