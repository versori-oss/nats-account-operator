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
	accountsnatsiov1alpha1 "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
)

// AccountReconciler reconciles a Account object
type AccountReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	CV1Interface      corev1.CoreV1Interface
	AccountsClientSet accountsclientsets.AccountsV1alpha1Interface
	NatsClient        *NatsClient
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

	acc := new(accountsnatsiov1alpha1.Account)
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
		if err != nil {
			return
		}
		if !equality.Semantic.DeepEqual(originalStatus, acc.Status) {
			if err = r.Status().Update(ctx, acc); err != nil {
				if errors.IsConflict(err) {
					result.RequeueAfter = time.Second * 5
					return
				}
				logger.Error(err, "failed to update account status")

			}
		}
	}()

	sKeys, err := r.ensureSigningKeysUpdated(ctx, acc)
	if err != nil {
		logger.Error(err, "failed to ensure signing keys were updated")
		return ctrl.Result{}, err
	}

	if err := r.ensureOperatorResolved(ctx, acc); err != nil {
		logger.Error(err, "failed to ensure the operator owning the account was resolved")
		return ctrl.Result{}, err
	}

	if err := r.ensureSeedJWTSecrets(ctx, acc, sKeys); err != nil {
		logger.Error(err, "failed to ensure account jwt secret")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AccountReconciler) ensureSigningKeysUpdated(ctx context.Context, acc *accountsnatsiov1alpha1.Account) ([]accountsnatsiov1alpha1.SigningKeyEmbeddedStatus, error) {
	logger := log.FromContext(ctx)

	skList, err := r.AccountsClientSet.SigningKeys(acc.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil && len(skList.Items) == 0 {
		logger.Info("no signing keys found")
		acc.Status.MarkSigningKeysUpdateUnknown("no signing keys found", "")
		return nil, nil
	} else if err != nil {
		logger.Error(err, "failed to list signing keys")
		return nil, err
	}

	signingKeys := make([]accountsnatsiov1alpha1.SigningKeyEmbeddedStatus, 0)
	for _, sk := range skList.Items {
		if sk.Status.OwnerRef.Namespace == acc.Namespace && sk.Status.OwnerRef.Name == acc.Name {
			signingKeys = append(signingKeys, accountsnatsiov1alpha1.SigningKeyEmbeddedStatus{
				Name: sk.Name,
				KeyPair: accountsnatsiov1alpha1.KeyPair{
					PublicKey:      sk.Status.KeyPair.PublicKey,
					SeedSecretName: sk.Status.KeyPair.SeedSecretName,
				},
			})
		}
	}

	if len(signingKeys) == 0 {
		logger.Info("no signing keys found for account")
		acc.Status.MarkSigningKeysUpdateUnknown("no signing keys found for account", "account: %s", acc.Name)
		return nil, nil
	}

	acc.Status.MarkSigningKeysUpdated(signingKeys)
	return signingKeys, nil
}

func (r *AccountReconciler) ensureOperatorResolved(ctx context.Context, acc *accountsnatsiov1alpha1.Account) error {
	// logger = log.FromContext(ctx)

	// operators, err := r.AccountsClientSet.Operators(acc.Namespace).List(ctx, metav1.ListOptions{})

	// WTF AM I DOING HERE?

	opRef := accountsnatsiov1alpha1.InferredObjectReference{
		Name:      "operator-test",
		Namespace: "nats-accounts-operator-system",
	}
	acc.Status.MarkOperatorResolved(opRef)

	return nil
}

func (r *AccountReconciler) ensureSeedJWTSecrets(ctx context.Context, acc *accountsnatsiov1alpha1.Account, sKeys []accountsnatsiov1alpha1.SigningKeyEmbeddedStatus) error {
	logger := log.FromContext(ctx)

	_, errSeed := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, acc.Spec.SeedSecretName, metav1.GetOptions{})
	jwtSec, errJWT := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, acc.Spec.JWTSecretName, metav1.GetOptions{})

	var sKeysPublicKeys []string
	for _, sk := range sKeys {
		sKeysPublicKeys = append(sKeysPublicKeys, sk.KeyPair.PublicKey)
	}

	// if one or the other does not exist, then create them both
	if errors.IsNotFound(errSeed) || errors.IsNotFound(errJWT) {
		// find the signing key from the spec and verify it belongs to an operator.

		// TODO @JoeLanglands might be better to do this in the sameway that a signing key resolves its owner. See signing key controller ensureOwnerResolved
		// Check ensureAccountResolved in user_controller. This should form the basis of ensureOperatorResolved too!
		sKey, err := r.AccountsClientSet.SigningKeys(acc.Namespace).Get(ctx, acc.Spec.SigningKey.Ref.Name, metav1.GetOptions{})
		if err != nil {
			logger.Info("failed to get signing key", "signing key", acc.Spec.SigningKey.Ref.Name)
			acc.Status.MarkJWTPushFailed("failed to get signing key", "spec signing key: %s", acc.Spec.SigningKey.Ref.Name)
			acc.Status.MarkSeedSecretFailed("failed to get signing key", "spec signing key: %s", acc.Spec.SigningKey.Ref.Name)
			return nil
		}
		if sKey.Status.OwnerRef.Kind != "Operator" {
			logger.Info("signing key is not owned by an operator")
			return errors.NewBadRequest("signing key is not owned by an operator")
		}

		// TODO @JoeLanglands: ensure that this account can be managed by the operator that owns the signing key.
		// this involves following the namespace and label selector restrictions.

		accClaims := jwt.Account{
			Imports:     convertToNATSImports(acc.Spec.Imports),
			Exports:     convertToNATSExports(acc.Spec.Exports),
			Identities:  convertToNATSIdentities(acc.Spec.Identities),
			Limits:      convertToNATSOperatorLimits(acc.Spec.Limits),
			SigningKeys: sKeysPublicKeys,
		}

		skSeedSecret, err := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, sKey.Status.KeyPair.SeedSecretName, metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "failed to get seed secret for signing key")
			return err
		}

		kPair, err := nkeys.FromSeed(skSeedSecret.Data["seed"])
		if err != nil {
			logger.Error(err, "failed to make key pair from seed")
			return err
		}

		ajwt, publicKey, seed, err := CreateAccount(acc.Name, accClaims, kPair)
		if err != nil {
			logger.Error(err, "failed to create account")
			return err
		}

		jwtSecret := NewSecret(acc.Spec.JWTSecretName, acc.Namespace, WithData(map[string][]byte{"jwt": []byte(ajwt)}), WithImmutable(true))
		if err := ctrl.SetControllerReference(acc, &jwtSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set account as owner of jwt secret")
			return err
		}
		if _, err = r.CV1Interface.Secrets(acc.Namespace).Create(ctx, &jwtSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create jwt secret")
			return err
		}

		seedSecret := NewSecret(acc.Spec.SeedSecretName, acc.Namespace, WithData(map[string][]byte{"seed": seed}), WithImmutable(true))
		if err := ctrl.SetControllerReference(acc, &seedSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set account as owner of seed secret")
			return err
		}
		if _, err = r.CV1Interface.Secrets(acc.Namespace).Create(ctx, &seedSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create seed secret")
			return err
		}

		err = r.NatsClient.PushAccountJWT(ctx, ajwt)
		if err != nil {
			logger.Info("failed to push account jwt to nats server", "error", err)
			acc.Status.MarkJWTPushFailed("failed to push account jwt to nats server", "error: %s", err)
			return nil
		}

		acc.Status.MarkJWTPushed()
		acc.Status.MarkJWTSecretReady()
		acc.Status.MarkSeedSecretReady(publicKey, seedSecret.Name)
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
		// check if signing keys have been updated, if so update the jwt with the new signing keys
		accJWTEncoded := string(jwtSec.Data["jwt"])
		accClaims, err := jwt.DecodeAccountClaims(accJWTEncoded)
		if err != nil {
			logger.Error(err, "failed to decode account jwt")
			acc.Status.MarkJWTPushUnknown("failed to decode account jwt", "error: %s", err)
			return err
		}

		// if accClaims.SigningKeys

		// if we get here then the seed and jwts secrets should exist, check if the jwt has been pushed to NATS
		jwt, err := r.NatsClient.GetAccountJWT(ctx, acc.Status.KeyPair.PublicKey)
		if err != nil {
			// something else is wrong, so report the error
			logger.Error(err, "failed to get account jwt from nats server")
			acc.Status.MarkJWTPushUnknown("failed to get account jwt from nats server", "error: %s", err)
			return err
		}
		if jwt == "" {
			// jwt isn't pushed to NATS, try to push it
			err2 := r.NatsClient.PushAccountJWT(ctx, string(jwtSec.Data["jwt"]))
			if err2 != nil {
				logger.Error(err2, "failed to push account jwt to nats server")
				acc.Status.MarkJWTPushFailed("failed to push account jwt to nats server", "error: %s", err)
				return err2
			}
		}
		acc.Status.MarkJWTPushed()
		acc.Status.MarkSeedSecretReady(acc.Status.KeyPair.PublicKey, acc.Status.KeyPair.SeedSecretName)
		acc.Status.MarkJWTSecretReady()
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("AccountReconciler")
	err := ctrl.NewControllerManagedBy(mgr).
		For(&accountsnatsiov1alpha1.Account{}).
		Owns(&v1.Secret{}).
		Watches(
			&source.Kind{Type: &accountsnatsiov1alpha1.SigningKey{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				signingKey, ok := obj.(*accountsnatsiov1alpha1.SigningKey)
				if !ok {
					logger.Info("SigningKey watcher received non-SigningKey object",
						"kind", obj.GetObjectKind().GroupVersionKind().String())
					return nil
				}

				ownerRef := signingKey.Status.OwnerRef
				if ownerRef == nil {
					return nil
				}

				accountGVK := accountsnatsiov1alpha1.GroupVersion.WithKind("Account")
				if accountGVK != ownerRef.GetGroupVersionKind() {
					logger.V(1).Info("SigningKey watcher received SigningKey with non-account owner",
						"owner", ownerRef.GetGroupVersionKind().String())
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
