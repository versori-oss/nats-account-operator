package resources

import (
    "github.com/nats-io/jwt/v2"
    "github.com/nats-io/nkeys"
    "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type UserCredentialSecretBuilder struct {
    scheme *runtime.Scheme
    secret *corev1.Secret
}

func NewUserCredentialSecretBuilder(scheme *runtime.Scheme) *UserCredentialSecretBuilder {
    return &UserCredentialSecretBuilder{
        scheme: scheme,
        secret: &corev1.Secret{},
    }
}

func NewUserCredentialSecretBuilderFromSecret(s *corev1.Secret, scheme *runtime.Scheme) *UserCredentialSecretBuilder {
    return &UserCredentialSecretBuilder{
        scheme: scheme,
        secret: s,
    }
}

func (b *UserCredentialSecretBuilder) Build(usr *v1alpha1.User, ujwt string, seed []byte) (*corev1.Secret, error) {
    creds, err := jwt.FormatUserConfig(ujwt, seed)
    if err != nil {
        return nil, err
    }

    var pubkey string

    // we don't care about errors, pubkey can be empty however improbably that is, we have bigger problems upstream if
    // this fails
    if kp, err := nkeys.FromSeed(seed); err == nil {
        pubkey, _ = kp.PublicKey()
    }

    if b.secret.Annotations == nil {
        b.secret.Annotations = make(map[string]string)
    }

    if b.secret.Labels == nil {
        b.secret.Labels = make(map[string]string)
    }

    b.secret.Name = usr.Spec.CredentialsSecretName
    b.secret.Labels[LabelJWTSubject] = pubkey
    b.secret.Annotations[AnnotationSecretJWTType] = AnnotationSecretTypeUser
    b.secret.Namespace = usr.GetNamespace()
    b.secret.Data = map[string][]byte{
        v1alpha1.NatsSecretCredsKey:      creds,
    }

    if err = controllerutil.SetControllerReference(usr, b.secret, b.scheme); err != nil {
        return nil, err
    }

    return b.secret, nil
}
