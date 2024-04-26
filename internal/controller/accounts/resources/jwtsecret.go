package resources

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

type JWTSecretBuilder struct {
	scheme *runtime.Scheme
	secret *v1.Secret
}

func NewJWTSecretBuilder(scheme *runtime.Scheme) *JWTSecretBuilder {
	return &JWTSecretBuilder{
		scheme: scheme,
		secret: &v1.Secret{},
	}
}

func NewJWTSecretBuilderFromSecret(s *v1.Secret, scheme *runtime.Scheme) *JWTSecretBuilder {
	return &JWTSecretBuilder{
		scheme: scheme,
		secret: s.DeepCopy(),
	}
}

func (b *JWTSecretBuilder) Build(obj client.Object, jwt string, opts ...SecretOption) (*v1.Secret, error) {
	for _, opt := range opts {
		if err := opt(b.secret); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if b.secret.Labels == nil {
		b.secret.Labels = make(map[string]string)
	}

	b.secret.Labels[LabelSecretType] = LabelSecretTypeJWT

	switch v := obj.(type) {
	case *v1alpha1.Operator:
		b.secret.Name = v.Spec.JWTSecretName
		b.secret.Labels[LabelSecretJWTType] = LabelSecretTypeOperator
		b.secret.Labels[LabelOperatorName] = v.Name
	case *v1alpha1.Account:
		b.secret.Name = v.Spec.JWTSecretName
		b.secret.Labels[LabelSecretJWTType] = LabelSecretTypeAccount
		b.secret.Labels[LabelAccountName] = v.Name
	case *v1alpha1.User:
		b.secret.Name = v.Spec.JWTSecretName
		b.secret.Labels[LabelSecretJWTType] = LabelSecretTypeUser
		b.secret.Labels[LabelUserName] = v.Name
	default:
		return nil, fmt.Errorf("unknown object type for JWT secret owner: %T", obj)
	}

	b.secret.Namespace = obj.GetNamespace()
	b.secret.Data = map[string][]byte{
		v1alpha1.NatsSecretJWTKey: []byte(jwt),
	}

	if err := controllerutil.SetControllerReference(obj, b.secret, b.scheme); err != nil {
		return nil, fmt.Errorf("failed to set owner reference: %w", err)
	}

	return b.secret, nil
}
