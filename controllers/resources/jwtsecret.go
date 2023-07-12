package resources

import (
	"fmt"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		secret: s,
	}
}

func (b *JWTSecretBuilder) Build(obj client.Object, jwt string) (*v1.Secret, error) {
	if b.secret.Annotations == nil {
		b.secret.Annotations = make(map[string]string)
	}

	switch v := obj.(type) {
	case *v1alpha1.Account:
		b.secret.Name = v.Spec.JWTSecretName
		b.secret.Annotations[AnnotationSecretJWTType] = AnnotationSecretTypeAccount
	case *v1alpha1.User:
		b.secret.Name = v.Spec.JWTSecretName
		b.secret.Annotations[AnnotationSecretJWTType] = AnnotationSecretTypeUser
	default:
		return nil, fmt.Errorf("unknown object type for JWT secret owner: %T", obj)
	}

	b.secret.Namespace = obj.GetNamespace()
	b.secret.Data = map[string][]byte{
		v1alpha1.NatsSecretJWTKey: []byte(jwt),
	}

	return b.secret, nil
}
