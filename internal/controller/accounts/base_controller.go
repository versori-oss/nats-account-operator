package controllers

import (
	"context"
	"fmt"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	"github.com/versori-oss/nats-account-operator/internal/controller/accounts/resources"
	accountsv1alpha1 "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
	"github.com/versori-oss/nats-account-operator/pkg/nsc"
)

type NKeyFactory func() (nkeys.KeyPair, error)

type BaseReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	CoreV1           corev1.CoreV1Interface
	AccountsV1Alpha1 accountsv1alpha1.AccountsV1alpha1Interface
	EventRecorder    record.EventRecorder
}

func (r *BaseReconciler) reconcileSeedSecret(ctx context.Context, owner client.Object, newKP NKeyFactory, secretName string, secretOpts ...resources.SecretOption) (*v1alpha1.KeyPair, []byte, error) {
	logger := log.FromContext(ctx)

	got, err := r.CoreV1.Secrets(owner.GetNamespace()).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("seed secret does not exist, creating")

			return r.createSeedSecret(ctx, owner, newKP, secretOpts...)
		}

		return nil, nil, TemporaryError(ConditionUnknown(v1alpha1.ReasonUnknownError, "failed to get seed secret: %w", err))
	}

	logger.V(2).Info("found existing seed secret, ensuring it is up to date")

	kp, err := r.ensureSeedSecretUpToDate(ctx, owner, got, secretOpts...)

	return kp, got.Data[v1alpha1.NatsSecretSeedKey], err
}

func (r *BaseReconciler) ensureSeedSecretUpToDate(ctx context.Context, owner client.Object, got *v1.Secret, secretOpts ...resources.SecretOption) (*v1alpha1.KeyPair, error) {
	logger := log.FromContext(ctx)

	seed, ok := got.Data[v1alpha1.NatsSecretSeedKey]
	if !ok {
		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonInvalidSeedSecret, "seed secret does not contain seed data, delete the secret for a new keypair"))
	}

	kp, err := nkeys.FromSeed(seed)
	if err != nil {
		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonInvalidSeedSecret, "failed to parse seed: %w", err))
	}

	pubkey, err := kp.PublicKey()
	if err != nil {
		// this shouldn't really happen if everything is stitched up correctly, it probably means something has
		// been manually changed in the secret to an invalid seed.
		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonMalformedSeedSecret, "failed to get PublicKey from KeyPair: %w", err))
	}

	want, err := resources.NewKeyPairSecretBuilderFromSecret(got, r.Scheme).Build(owner, kp, secretOpts...)
	if err != nil {
		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to build seed secret from existing secret: %w", err))
	}

	if !equality.Semantic.DeepEqual(got, want) {
		logger.V(1).Info("seed secret does not match desired state, updating")

		err = r.Client.Update(ctx, want)
		if err != nil {
			return nil, TemporaryError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to update seed secret: %w", err))
		}

		r.EventRecorder.Eventf(owner, v1.EventTypeNormal, "SeedSecretUpdated", "updated secret: %s/%s", want.Namespace, want.Name)
	}

	return &v1alpha1.KeyPair{
		PublicKey:      pubkey,
		SeedSecretName: got.Name,
	}, nil
}

func (r *BaseReconciler) createSeedSecret(ctx context.Context, obj client.Object, newKP NKeyFactory, secretOpts ...resources.SecretOption) (*v1alpha1.KeyPair, []byte, error) {
	kp, err := newKP()
	if err != nil {
		return nil, nil, TemporaryError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to generate new KeyPair: %w", err))
	}

	seed, err := kp.Seed()
	if err != nil {
		return nil, nil, TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to get Seed from nkey.KeyPair: %w", err))
	}

	pubkey, err := kp.PublicKey()
	if err != nil {
		return nil, nil, TerminalError(ConditionFailed(
			v1alpha1.ReasonMalformedSeedSecret, "failed to get PublicKey from nkey.KeyPair: %w", err))
	}

	secret, err := resources.NewKeyPairSecretBuilder(r.Scheme).Build(obj, kp, secretOpts...)
	if err != nil {
		return nil, nil, TemporaryError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to build seed secret: %w", err))
	}

	if err = r.Client.Create(ctx, secret); err != nil {
		return nil, nil, TemporaryError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to create seed secret: %w", err))
	}

	r.EventRecorder.Eventf(obj, v1.EventTypeNormal, "SeedSecretCreated", "created secret: %s/%s", secret.Namespace, secret.Name)

	return &v1alpha1.KeyPair{
		PublicKey:      pubkey,
		SeedSecretName: secret.Name,
	}, seed, nil
}

func (r *BaseReconciler) createJWTSecret(ctx context.Context, obj client.Object, jwt string) error {
	secret, err := resources.NewJWTSecretBuilder(r.Scheme).Build(obj, jwt)
	if err != nil {
		return TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to build account keypair secret: %w", err))
	}

	if err := r.Client.Create(ctx, secret); err != nil {
		return TemporaryError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to create account JWT secret: %w", err))
	}

	r.EventRecorder.Eventf(obj, v1.EventTypeNormal, "JWTSecretCreated", "created secret: %s/%s", secret.Namespace, secret.Name)

	return nil
}

// ensureJWTSecretUpToDate compares that the existing JWT secret decodes and matches the expected claims, if it does not
// match the secret will be updated with the nextJWT value.
func (r *BaseReconciler) ensureJWTSecretUpToDate(ctx context.Context, acc client.Object, wantClaims any, got *v1.Secret, nextJWT string) (string, error) {
	logger := log.FromContext(ctx)

	gotJWT, ok := got.Data[v1alpha1.NatsSecretJWTKey]
	if !ok {
		for _, ownerRef := range got.OwnerReferences {
			if ownerRef.UID == acc.GetUID() {
				logger.Info("existing JWT secret does not contain JWT data, deleting to generate a new JWT")

				err := r.Client.Delete(ctx, got)
				if err != nil {
					logger.Error(err, "failed to delete JWT secret")

					return "", TemporaryError(ConditionUnknown(v1alpha1.ReasonUnknownError, "failed to delete invalid JWT secret: %w", err))
				}

				r.EventRecorder.Eventf(acc, v1.EventTypeNormal, "JWTSecretDeleted", "deleted secret: %s/%s", got.Namespace, got.Name)

				return nextJWT, r.createJWTSecret(ctx, acc, nextJWT)
			}
		}

		return "", TerminalError(ConditionFailed(v1alpha1.ReasonInvalidJWTSecret, "JWT secret does not contain JWT data and is not owned by this controller"))
	}

	gotClaims, err := jwt.Decode(string(gotJWT))
	switch {
	case err != nil:
		logger.Info("failed to decode JWT from secret, updating to latest version", "reason", err.Error())
	case !nsc.Equality.DeepEqual(gotClaims, wantClaims):
		logger.V(1).Info("existing JWT secret does not match desired claims, updating to latest version")
	default:
		logger.V(1).Info("existing JWT secret matches desired claims, no update required")

		return string(gotJWT), nil
	}

	want, err := resources.NewJWTSecretBuilderFromSecret(got, r.Scheme).Build(acc, nextJWT)
	if err != nil {
		return "", TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to build desired JWT secret: %w", err))
	}

	err = r.Client.Update(ctx, want)
	if err != nil {
		if errors.IsNotFound(err) {
			return nextJWT, r.createJWTSecret(ctx, acc, nextJWT)
		}

		logger.Error(err, "failed to update JWT secret")

		return "", TemporaryError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to update JWT secret: %w", err))
	}

	r.EventRecorder.Eventf(acc, v1.EventTypeNormal, "JWTSecretUpdated", "updated secret: %s/%s", want.Namespace, want.Name)

	return nextJWT, nil
}

// resolveIssuer resolves the issuer reference to a KeyPairable object. This is abstracted to support issuers being
// either a SigningKey, or an Operator/Account where the object being reconciled is an Account/User respectively.
//
// The returned bool is true when everything is ok, or if a temporary error has occurred and the
// reconciliation should be re-enqueued.
func (r *BaseReconciler) resolveIssuer(ctx context.Context, issuer v1alpha1.IssuerReference, fallbackNamespace string) (kp v1alpha1.KeyPairable, err error) {
	logger := log.FromContext(ctx)

	issuerGVK := issuer.Ref.GetGroupVersionKind()

	obj, err := r.Scheme.New(issuerGVK)
	if err != nil {
		logger.Error(err, "failed to create issuer object from scheme", "issuer_gvk", issuerGVK.String())

		return nil, TerminalError(ConditionFailed(
			v1alpha1.ReasonUnsupportedIssuer, "unsupported GroupVersionKind: %s", err.Error()))
	}

	issuerObj, ok := obj.(client.Object)
	if !ok {
		logger.Info("failed to convert runtime.Object to client.Object",
			"issuer_gvk", issuerGVK.String(),
			"obj_type", fmt.Sprintf("%T", obj),
		)

		return nil, TerminalError(ConditionFailed(
			v1alpha1.ReasonUnsupportedIssuer, "runtime.Object cannot be converted to client.Object", issuerGVK.String()))
	}

	// .issuer.ref.namespace is optional, so default to the Account's namespace if not set
	if issuer.Ref.Namespace != "" {
		fallbackNamespace = issuer.Ref.Namespace
	}

	err = r.Get(ctx, client.ObjectKey{
		Namespace: fallbackNamespace,
		Name:      issuer.Ref.Name,
	}, issuerObj)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, TemporaryError(ConditionFailed(
				v1alpha1.ReasonNotFound, "%s, %s/%s: not found", issuerGVK.String(), fallbackNamespace, issuer.Ref.Name))
		}

		return nil, ConditionUnknown(v1alpha1.ReasonUnknownError, "failed to get issuer object from client: %w", err)
	}

	keyPairable, ok := issuerObj.(v1alpha1.KeyPairable)
	if !ok {
		logger.Info("issuer does not implement KeyPairable interface", "issuer_type", fmt.Sprintf("%T", issuer))

		return nil, TerminalError(ConditionFailed(
			v1alpha1.ReasonUnsupportedIssuer, "issuer does not implement KeyPairable interface"))
	}

	conditions := keyPairable.GetConditionSet().Manage(keyPairable.GetStatus())

	// Initialize the conditions if they are not already set, not doing this causes a nil-pointer dereference panic
	conditions.InitializeConditions()

	readyCondition := conditions.GetTopLevelCondition()
	if !readyCondition.IsTrue() {
		logger.V(1).Info("issuer seed secret is not ready", "issuer_type", fmt.Sprintf("%T", issuer), "reason", readyCondition.Reason, "message", readyCondition.Message)

		return nil, TemporaryError(ConditionFailed(
			v1alpha1.ReasonNotReady, "issuer seed secret is not ready"))
	}

	return keyPairable, nil
}

func (r *BaseReconciler) resolveSigningKeyOwner(ctx context.Context, sk *v1alpha1.SigningKey) (client.Object, error) {
	logger := log.FromContext(ctx)

	if !sk.Status.GetCondition(v1alpha1.SigningKeyConditionOwnerResolved).IsTrue() {
		return nil, TemporaryError(ConditionUnknown(v1alpha1.ReasonNotReady, "signing key owner has not been resolved"))
	}

	gvk := sk.Status.OwnerRef.GetGroupVersionKind()

	obj, err := r.Scheme.New(gvk)
	if err != nil {
		logger.Error(err, "failed to create owner object from scheme", "owner_gvk", gvk.String())

		return nil, TemporaryError(ConditionFailed(
			v1alpha1.ReasonInvalidSigningKeyOwner, "unsupported GroupVersionKind: %s", err.Error()))
	}

	owner, ok := obj.(client.Object)
	if !ok {
		logger.Info("failed to convert runtime.Object to client.Object",
			"owner_gvk", gvk.String(),
			"owner_type", fmt.Sprintf("%T", obj),
		)

		return nil, TemporaryError(ConditionFailed(
			v1alpha1.ReasonInvalidSigningKeyOwner, "runtime.Object cannot be converted to client.Object", gvk.String()))
	}

	err = r.Client.Get(ctx, client.ObjectKey{
		Namespace: sk.Status.OwnerRef.Namespace,
		Name:      sk.Status.OwnerRef.Name,
	}, owner)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, TemporaryError(ConditionFailed(
				v1alpha1.ReasonNotFound, "%s, %s/%s: not found", gvk.String(), sk.Status.OwnerRef.Namespace, sk.Status.OwnerRef.Name))
		}

		return nil, ConditionUnknown(v1alpha1.ReasonUnknownError, "failed to get owner object from client: %w", err)
	}

	return owner, nil
}

func (r *BaseReconciler) loadIssuerSeed(ctx context.Context, issuer v1alpha1.KeyPairable, wantPrefix nkeys.PrefixByte) (nkeys.KeyPair, error) {
	logger := log.FromContext(ctx)

	keyPair := issuer.GetKeyPair()
	if keyPair == nil {
		logger.Info("WARNING! issuer KeyPair is nil, but condition checks should have caught this")

		return nil, ConditionFailed(v1alpha1.ReasonUnknownError, "issuer KeyPair is nil")
	}

	skSeedSecret, err := r.CoreV1.Secrets(issuer.GetNamespace()).Get(ctx, keyPair.SeedSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, TemporaryError(ConditionFailed(
				v1alpha1.ReasonNotFound, "core/v1; Secret, %s/%s: not found", issuer.GetNamespace(), issuer.GetName()))
		}

		logger.V(1).Info("failed to get issuer seed", "issuer", issuer.GetName())

		return nil, ConditionUnknown(v1alpha1.ReasonIssuerSeedError, "failed to get issuer seed: %s", err.Error())
	}

	seed, ok := skSeedSecret.Data[v1alpha1.NatsSecretSeedKey]
	if !ok {
		// TODO: this is a terminal error, but if the secret is updated, this will only trigger a
		//  reconcile on the owning issuer, and not the user.
		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonMalformedSeedSecret, "secret missing required field: %s", v1alpha1.NatsSecretSeedKey))
	}

	prefix, _, err := nkeys.DecodeSeed(seed)
	if err != nil {
		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonMalformedSeedSecret, "failed to parse seed: %s", err.Error()))
	}

	if prefix != wantPrefix {
		return nil, TerminalError(ConditionFailed(
			v1alpha1.ReasonMalformedSeedSecret,
			"unexpected seed prefix, wanted %q but got %q",
			wantPrefix.String(),
			prefix.String(),
		))
	}

	// we've already decoded the seed once to check the prefix, so we can ignore this error
	kp, _ := nkeys.FromSeed(seed)

	pk, err := kp.PublicKey()
	if err != nil {
		logger.Error(err, "failed to get public key from seed")

		return nil, TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to get public key from seed: %s", err.Error()))
	}

	// check that the public key generated from the secret matches the public key in the issuer's KeyPair status, if
	// this fails then the issuer is probably going to reconcile again soon, and we'll be enqueued again afterwards.
	if pk != keyPair.PublicKey {
		return nil, TerminalError(ConditionFailed(
			v1alpha1.ReasonPublicKeyMismatch,
			"public key mismatch, wanted %q but got %q",
			keyPair.PublicKey,
			pk,
		))
	}

	return kp, nil
}
