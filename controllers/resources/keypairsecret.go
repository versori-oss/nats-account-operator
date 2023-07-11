package resources

import (
    "fmt"
    "github.com/nats-io/nkeys"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		secret: s,
	}
}

func (b *KeyPairSecretBuilder) Build(obj client.Object, kp nkeys.KeyPair) (*v1.Secret, error) {
	seed, err := kp.Seed()
	if err != nil {
		return nil, err
	}

	pubkey, err := kp.PublicKey()
	if err != nil {
		return nil, err
	}

    if b.secret.Annotations == nil {
        b.secret.Annotations = make(map[string]string)
    }

    switch v := obj.(type) {
    case *v1alpha1.Account:
        b.secret.Name = v.Spec.SeedSecretName
        b.secret.Annotations[AnnotationSecretJWTType] = AnnotationSecretTypeAccount
    case *v1alpha1.User:
        b.secret.Name = v.Spec.SeedSecretName
        b.secret.Annotations[AnnotationSecretJWTType] = AnnotationSecretTypeUser
    default:
        return nil, fmt.Errorf("unknown object type for JWT secret owner: %T", obj)
    }

	b.secret.Namespace = obj.GetNamespace()
	b.secret.Data = map[string][]byte{
		v1alpha1.NatsSecretSeedKey:      seed,
		v1alpha1.NatsSecretPublicKeyKey: []byte(pubkey),
	}

	if err = controllerutil.SetControllerReference(obj, b.secret, b.scheme); err != nil {
		return nil, err
	}

	return b.secret, nil
}
