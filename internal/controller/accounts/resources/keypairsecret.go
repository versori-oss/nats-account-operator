package resources

import (
	"fmt"

	"github.com/nats-io/nkeys"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

type KeyPairSecretBuilder struct {
	scheme *runtime.Scheme
	secret *v1.Secret
}

func NewKeyPairSecretBuilder(scheme *runtime.Scheme) *KeyPairSecretBuilder {
	return &KeyPairSecretBuilder{
		scheme: scheme,
		secret: &v1.Secret{},
	}
}

func NewKeyPairSecretBuilderFromSecret(s *v1.Secret, scheme *runtime.Scheme) *KeyPairSecretBuilder {
	return &KeyPairSecretBuilder{
		scheme: scheme,
		secret: s.DeepCopy(),
	}
}

func (b *KeyPairSecretBuilder) Build(obj client.Object, kp nkeys.KeyPair, opts ...SecretOption) (*v1.Secret, error) {
	for _, opt := range opts {
		if err := opt(b.secret); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	seed, err := kp.Seed()
	if err != nil {
		return nil, err
	}

	pubkey, err := kp.PublicKey()
	if err != nil {
		return nil, err
	}

	if b.secret.Labels == nil {
		b.secret.Labels = make(map[string]string)
	}

	b.secret.Labels[LabelSecretType] = LabelSecretTypeSeed
	b.secret.Labels[LabelSubject] = pubkey

	switch v := obj.(type) {
	case *v1alpha1.Operator:
		b.secret.Name = v.Spec.SeedSecretName
		b.secret.Labels[LabelSecretSeedType] = LabelSecretTypeOperator
		b.secret.Labels[LabelOperatorName] = v.Name
	case *v1alpha1.SigningKey:
		b.secret.Name = v.Spec.SeedSecretName
		b.secret.Labels[LabelSecretSeedType] = LabelSecretTypeSigningKey
		b.secret.Labels[LabelSigningKeyName] = v.Name
	case *v1alpha1.Account:
		b.secret.Name = v.Spec.SeedSecretName
		b.secret.Labels[LabelSecretSeedType] = LabelSecretTypeAccount
		b.secret.Labels[LabelAccountName] = v.Name
	case *v1alpha1.User:
		b.secret.Name = v.Spec.SeedSecretName
		b.secret.Labels[LabelSecretSeedType] = LabelSecretTypeUser
		b.secret.Labels[LabelUserName] = v.Name
	default:
		return nil, fmt.Errorf("unknown object type for JWT secret owner: %T", obj)
	}

	b.secret.Namespace = obj.GetNamespace()
	b.secret.Data = map[string][]byte{
		v1alpha1.NatsSecretSeedKey:      seed,
		v1alpha1.NatsSecretPublicKeyKey: []byte(pubkey),
	}

	if err := controllerutil.SetControllerReference(obj, b.secret, b.scheme); err != nil {
		return nil, fmt.Errorf("failed to set owner reference: %w", err)
	}

	return b.secret, nil
}
