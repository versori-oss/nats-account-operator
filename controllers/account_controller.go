/*
MIT License

Copyright (c) 2022 Versori Ltd

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package controllers

import (
	"context"
	"fmt"
	"github.com/versori-oss/nats-account-operator/pkg/apis"
	"time"

	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nkeys"
	v1alpha1 "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
	"github.com/versori-oss/nats-account-operator/pkg/nsc"
)

// AccountReconciler reconciles a Account object
type AccountReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	CV1Interface      corev1.CoreV1Interface
	AccountsClientSet accountsclientsets.AccountsV1alpha1Interface
}

//+kubebuilder:rbac:groups=accounts.nats.io,resources=accounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=accounts.nats.io,resources=accounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=accounts.nats.io,resources=accounts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Account object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *AccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("reconciling account", "account", req.Name)

	acc := new(v1alpha1.Account)
	if err := r.Client.Get(ctx, req.NamespacedName, acc); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("account deleted")
			return ctrl.Result{}, nil
		}

		logger.Error(err, "failed to get account")
		return ctrl.Result{}, err
	}

	originalStatus := acc.Status.DeepCopy()

	defer func() {
		if !equality.Semantic.DeepEqual(originalStatus, acc.Status) {
			if err2 := r.Status().Update(ctx, acc); err2 != nil {
				if errors.IsConflict(err2) {
					result.RequeueAfter = time.Second * 30

					err = nil
					return
				}

				logger.Error(err, "failed to update account status")
			}
		}
	}()

	opSKey, err := r.ensureOperatorResolved(ctx, acc)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("operator owner not found, requeuing")
			result.RequeueAfter = time.Second * 30
			return
		}
		logger.Error(err, "failed to ensure the operator owning the account was resolved")
		return ctrl.Result{}, err
	}

	sKeys, err := r.ensureSigningKeysUpdated(ctx, acc)
	if err != nil {
		if errors.IsNotFound(err) {
			result.RequeueAfter = time.Second * 30
			return
		}
		logger.Error(err, "failed to ensure signing keys were updated")
		return ctrl.Result{}, err
	}

	if err := r.ensureSeedJWTSecrets(ctx, acc, sKeys, opSKey); err != nil {
		logger.Error(err, "failed to ensure account jwt secret")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AccountReconciler) ensureSigningKeysUpdated(ctx context.Context, acc *v1alpha1.Account) ([]v1alpha1.SigningKeyEmbeddedStatus, error) {
	logger := log.FromContext(ctx)

	skList, err := r.AccountsClientSet.SigningKeys(acc.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil && len(skList.Items) == 0 {
		logger.Info("no signing keys found")
		acc.Status.MarkSigningKeysUpdateUnknown("no signing keys found", "")
		return nil, errors.NewNotFound(v1alpha1.Resource(v1alpha1.SigningKey{}.ResourceVersion), "signingkeys")
	} else if err != nil {
		logger.Error(err, "failed to list signing keys")
		return nil, err
	}

	signingKeys := make([]v1alpha1.SigningKeyEmbeddedStatus, 0)
	for _, sk := range skList.Items {
		if sk.Status.IsReady() && sk.Status.OwnerRef.Namespace == acc.Namespace && sk.Status.OwnerRef.Name == acc.Name {
			signingKeys = append(signingKeys, v1alpha1.SigningKeyEmbeddedStatus{
				Name:    sk.GetName(),
				KeyPair: *sk.Status.KeyPair,
			})
		}
	}

	if len(signingKeys) == 0 {
		logger.V(1).Info("no ready signing keys found for account")
		acc.Status.MarkSigningKeysUpdateUnknown("no ready signing keys found for account", "account: %s", acc.Name)
		return nil, errors.NewNotFound(v1alpha1.Resource(v1alpha1.SigningKey{}.ResourceVersion), "signingkeys")
	}

	acc.Status.MarkSigningKeysUpdated(signingKeys)
	return signingKeys, nil
}

func (r *AccountReconciler) ensureOperatorResolved(ctx context.Context, acc *v1alpha1.Account) ([]byte, error) {
	logger := log.FromContext(ctx)

	skRef := acc.Spec.SigningKey.Ref
	obj, err := r.Scheme.New(skRef.GetGroupVersionKind())
	if err != nil {
		acc.Status.MarkOperatorResolveFailed(
			v1alpha1.ReasonUnsupportedSigningKey, "unsupported GroupVersionKind: %s", err.Error())

		return nil, err
	}

	signingKey, ok := obj.(client.Object)
	if !ok {
		acc.Status.MarkOperatorResolveFailed(
			v1alpha1.ReasonUnsupportedSigningKey, "runtime.Object cannot be converted to client.Object", obj.GetObjectKind().GroupVersionKind())

		return nil, fmt.Errorf("failed to cast runtime.Object to client.Object")
	}

	err = r.Client.Get(ctx, client.ObjectKey{
		Namespace: skRef.Namespace,
		Name:      skRef.Name,
	}, signingKey)
	if err != nil {
		if errors.IsNotFound(err) {
			acc.Status.MarkOperatorResolveFailed(
				v1alpha1.ReasonNotFound,
				"%s, %s/%s: not found",
				signingKey.GetObjectKind().GroupVersionKind(),
				skRef.Namespace,
				skRef.Name,
			)

			return nil, err
		}

		acc.Status.MarkOperatorResolveUnknown(v1alpha1.ReasonUnknownError, "%s", err.Error())

		return nil, err
	}

	conditionAccessor, ok := signingKey.(apis.ConditionManagerAccessor)
	if !ok {
		acc.Status.MarkOperatorResolveFailed(
			v1alpha1.ReasonUnsupportedSigningKey,
			"%s does not implement ConditionAccessor: %T",
			signingKey.GetObjectKind().GroupVersionKind(),
			signingKey,
		)

		return nil, fmt.Errorf("signing key ref does not implement ConditionAccessor: %T", signingKey)
	}

	if !conditionAccessor.GetConditionManager().IsHappy() {
		acc.Status.MarkOperatorResolveUnknown(v1alpha1.ReasonNotReady, "signing key not ready")

		return nil, fmt.Errorf("signing key not ready")
	}

	var operator *v1alpha1.Operator

	switch owner := signingKey.(type) {
	case *v1alpha1.Operator:
		operator = owner
	case *v1alpha1.SigningKey:
		// this should only error if a network error occurs, we've checked that the signing key is happy so
		// owner refs should be set.
		operator, err = r.lookupOperatorForSigningKey(ctx, owner)
		if err != nil {
			return nil, err
		}
	default:
		acc.Status.MarkOperatorResolveFailed(
			v1alpha1.ReasonUnsupportedSigningKey,
			"expected Operator or SigningKey but got: %T",
			signingKey,
		)

		return nil, fmt.Errorf("expected Operator or SigningKey but got: %T", signingKey)
	}

	acc.Status.MarkOperatorResolved(v1alpha1.InferredObjectReference{
		Namespace: operator.Namespace,
		Name:      operator.Name,
	})

	keyPaired, ok := signingKey.(v1alpha1.KeyPairAccessor)
	if !ok {
		return nil, fmt.Errorf("signing key ref does not implement KeyPairAccessor: %T", signingKey)
	}

	keyPair := keyPaired.GetKeyPair()
	if keyPair == nil {
		return nil, fmt.Errorf("signing key ref does not have a key pair")
	}

	skSeedSecret, err := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, keyPair.SeedSecretName, metav1.GetOptions{})
	if err != nil {
		logger.Info("failed to get operator seed for signing key", "signing key: %s", signingKey.GetName(), "operator: %s", operator.GetName())
		return []byte{}, err
	}

	return skSeedSecret.Data[v1alpha1.NatsSecretSeedKey], nil
}

func (r *AccountReconciler) ensureSeedJWTSecrets(ctx context.Context, acc *v1alpha1.Account, sKeys []v1alpha1.SigningKeyEmbeddedStatus, opSkey []byte) error {
	logger := log.FromContext(ctx)

	natsHelper := nsc.NscHelper{
		OperatorRef:  acc.Status.OperatorRef,
		CV1Interface: r.CV1Interface,
		AccClientSet: r.AccountsClientSet,
	}

	_, errSeed := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, acc.Spec.SeedSecretName, metav1.GetOptions{})
	jwtSec, errJWT := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, acc.Spec.JWTSecretName, metav1.GetOptions{})

	var sKeysPublicKeys []string
	for _, sk := range sKeys {
		sKeysPublicKeys = append(sKeysPublicKeys, sk.KeyPair.PublicKey)
	}

	if errors.IsNotFound(errSeed) || errors.IsNotFound(errJWT) {
		accClaims := jwt.Account{
			Imports:     nsc.ConvertToNATSImports(acc.Spec.Imports),
			Exports:     nsc.ConvertToNATSExports(acc.Spec.Exports),
			Identities:  nsc.ConvertToNATSIdentities(acc.Spec.Identities),
			Limits:      nsc.ConvertToNATSOperatorLimits(acc.Spec.Limits),
			SigningKeys: sKeysPublicKeys,
		}

		kPair, err := nkeys.ParseDecoratedNKey(opSkey)
		if err != nil {
			logger.Error(err, "failed to make key pair from seed")
			return err
		}

		ajwt, publicKey, seed, err := nsc.CreateAccount(acc.Name, accClaims, kPair)
		if err != nil {
			logger.Error(err, "failed to create account")
			return err
		}

		jwtData := map[string][]byte{v1alpha1.NatsSecretJWTKey: []byte(ajwt)}
		jwtSecret := NewSecret(acc.Spec.JWTSecretName, acc.Namespace, WithData(jwtData), WithImmutable(false))
		if err := ctrl.SetControllerReference(acc, &jwtSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set account as owner of jwt secret")
			return err
		}

		if _, err := createOrUpdateSecret(ctx, r.CV1Interface, acc.Namespace, &jwtSecret, !errors.IsNotFound(errJWT)); err != nil {
			logger.Error(err, "failed to create or update jwt secret")
			return err
		}

		seedData := map[string][]byte{v1alpha1.NatsSecretSeedKey: seed, v1alpha1.NatsSecretPublicKeyKey: []byte(publicKey)}
		seedSecret := NewSecret(acc.Spec.SeedSecretName, acc.Namespace, WithData(seedData), WithImmutable(true))
		if err := ctrl.SetControllerReference(acc, &seedSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set account as owner of seed secret")
			return err
		}

		if _, err := createOrUpdateSecret(ctx, r.CV1Interface, acc.Namespace, &seedSecret, !errors.IsNotFound(errSeed)); err != nil {
			logger.Error(err, "failed to create or update seed secret")
			return err
		}

		if isSys, err := r.isSystemAccount(ctx, acc); err == nil && !isSys {
			err = natsHelper.PushJWT(ctx, ajwt)
			if err != nil {
				logger.Info("failed to push account jwt to nats server", "error", err)
				acc.Status.MarkJWTPushFailed("failed to push account jwt to nats server", "error: %s", err)
				return nil
			}
		} else if err != nil {
			logger.Error(err, "failed to determine if account is system account")
			return err
		}

		acc.Status.MarkJWTPushed()
		acc.Status.MarkJWTSecretReady()
		acc.Status.MarkSeedSecretReady(publicKey, seedSecret.Name)
		return nil
	} else if errSeed != nil {
		// logging and returning errors here since something could have actually gone wrong
		acc.Status.MarkSeedSecretUnknown("failed to get seed secret", "")
		logger.Error(errSeed, "failed to get seed secret")
		return errSeed
	} else if errJWT != nil {
		acc.Status.MarkJWTSecretUnknown("failed to get jwt secret", "")
		acc.Status.MarkJWTPushUnknown("failed to get jwt secret", "")
		logger.Error(errJWT, "failed to get jwt secret")
		return errJWT
	} else {
		err := r.updateAccountJWTSigningKeys(ctx, acc, opSkey, jwtSec, sKeysPublicKeys, &natsHelper)
		if err != nil {
			logger.V(1).Info("failed to update account JWT with signing keys", "error", err)
			acc.Status.MarkJWTPushUnknown("failed to update account JWT with signing keys", "")
			acc.Status.MarkJWTSecretUnknown("failed to update account JWT with signing keys", "")
			return nil
		}
	}

	return nil
}

func (r *AccountReconciler) updateAccountJWTSigningKeys(ctx context.Context, acc *v1alpha1.Account, operatorSeed []byte, jwtSecret *v1.Secret, sKeys []string, natsHelper nsc.NSCInterface) error {
	logger := log.FromContext(ctx)

	accJWTEncoded := string(jwtSecret.Data[v1alpha1.NatsSecretJWTKey])
	accClaims, err := jwt.DecodeAccountClaims(accJWTEncoded)
	if err != nil {
		logger.Error(err, "failed to decode account jwt")
		return err
	}

	if isEqualUnordered(accClaims.SigningKeys, sKeys) {
		logger.V(1).Info("account jwt signing keys are already up to date")
		return nil
	}

	accClaims.SigningKeys = jwt.StringList(sKeys)

	kPair, err := nkeys.ParseDecoratedNKey(operatorSeed)
	if err != nil {
		logger.Error(err, "failed to create nkeys from operator seed")
		return err
	}
	ajwt, err := accClaims.Encode(kPair)
	if err != nil {
		logger.Error(err, "failed to encode account jwt")
		return err
	}

	jwtSecret.Data[v1alpha1.NatsSecretJWTKey] = []byte(ajwt)
	_, err = r.CV1Interface.Secrets(jwtSecret.Namespace).Update(ctx, jwtSecret, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to update jwt secret")
		return err
	}

	if isSys, err := r.isSystemAccount(ctx, acc); err == nil && !isSys {
		err = natsHelper.UpdateJWT(ctx, accClaims.Subject, ajwt)
		if err != nil {
			logger.Error(err, "failed to update account jwt on nats server")
			return err
		}
	}

	return nil
}

// isSystemAccount returns true if acc is the system account this can only be used if the accounts operator has been resolved.
func (r *AccountReconciler) isSystemAccount(ctx context.Context, acc *v1alpha1.Account) (bool, error) {
	logger := log.FromContext(ctx)

	if !acc.Status.GetCondition(v1alpha1.AccountConditionOperatorResolved).IsTrue() {
		logger.V(1).Info("operator not resolved for account")
		// just return true here because we DO NOT push when true, so essentially wait for next reconcile when operator is resolved
		return true, nil
	}
	// the account is the system account if and only if the system account, in the operator CR referenced by account have the same name and namespace.
	operator, err := r.AccountsClientSet.Operators(acc.Status.OperatorRef.Namespace).Get(ctx, acc.Status.OperatorRef.Name, metav1.GetOptions{})
	if err != nil {
		return true, err
	}

	if !operator.Status.GetCondition(v1alpha1.OperatorConditionSystemAccountResolved).IsTrue() {
		logger.V(1).Info("system account not yet resolved for operator")
		return true, nil
	}

	if acc.Name == operator.Status.ResolvedSystemAccount.Name && acc.Namespace == operator.Status.ResolvedSystemAccount.Namespace {
		return true, nil
	}

	return false, nil
}

func (r *AccountReconciler) lookupOperatorForSigningKey(ctx context.Context, sk *v1alpha1.SigningKey) (*v1alpha1.Operator, error) {
	ownerRef := sk.Status.OwnerRef

	if ownerRef == nil {
		return nil, fmt.Errorf("signing key %s/%s has no owner reference", sk.Namespace, sk.Name)
	}

	return r.AccountsClientSet.Operators(ownerRef.Namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("AccountReconciler")
	err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Account{}).
		Owns(&v1.Secret{}).
		Watches(
			&source.Kind{Type: &v1alpha1.SigningKey{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				signingKey, ok := obj.(*v1alpha1.SigningKey)
				if !ok {
					logger.Info("SigningKey watcher received non-SigningKey object",
						"kind", obj.GetObjectKind().GroupVersionKind().String())
					return nil
				}

				ownerRef := signingKey.Status.OwnerRef
				if ownerRef == nil {
					return nil
				}

				accountGVK := v1alpha1.GroupVersion.WithKind("Account")
				if accountGVK != ownerRef.GetGroupVersionKind() {
					return nil
				}

				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name:      ownerRef.Name,
						Namespace: ownerRef.Namespace,
					},
				}}
			}),
		).
		Complete(r)

	if err != nil {
		return err
	}

	return nil
}
