package v1alpha1

// +k8s:deepcopy-gen=false
type KeyPairAccessor interface {
	GetKeyPair() *KeyPair
}

type KeyPair struct {
	PublicKey      string `json:"publicKey"`
	SeedSecretName string `json:"seedSecretName"`
}
