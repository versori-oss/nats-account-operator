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

	"go.uber.org/multierr"
	v1 "k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

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
			// logger.Info("updating acc status", "status", acc.Status, "originalStatus", originalStatus)
			if err = r.Status().Update(ctx, acc); err != nil {
				if errors.IsConflict(err) {
					result.RequeueAfter = time.Second * 5
					return
				}
				logger.Error(err, "failed to update account status")

			}
		}
	}()

	if err := r.ensureOperatorResolved(ctx, acc); err != nil {
		logger.Error(err, "failed to ensure the operator owning the account was resolved")
		return ctrl.Result{}, err
	}

	if err := r.ensureSeedJWTSecrets(ctx, acc); err != nil {
		logger.Error(err, "failed to ensure account jwt secret")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AccountReconciler) ensureOperatorResolved(ctx context.Context, acc *accountsnatsiov1alpha1.Account) error {
	_ = log.FromContext(ctx)

	// TODO @JoeLanglands implement me properly!

	opRef := accountsnatsiov1alpha1.InferredObjectReference{
		Name:      "operator-test",
		Namespace: "nats-accounts-operator-system",
	}
	acc.Status.MarkOperatorResolved(opRef)

	return nil
}

func (r *AccountReconciler) ensureSeedJWTSecrets(ctx context.Context, acc *accountsnatsiov1alpha1.Account) error {
	logger := log.FromContext(ctx)

	_, errSeed := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, acc.Spec.SeedSecretName, metav1.GetOptions{})
	_, errJWT := r.CV1Interface.Secrets(acc.Namespace).Get(ctx, acc.Spec.JWTSecretName, metav1.GetOptions{})

	// if one or the other does not exist, then create them both
	if errors.IsNotFound(errSeed) || errors.IsNotFound(errJWT) {
		// find the signing key from the spec and verify it belongs to an operator.
		sKey, err := r.AccountsClientSet.SigningKeys(acc.Namespace).Get(ctx, acc.Spec.SigningKey.Ref.Name, metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "failed to get signing key")
			return err
		}
		if sKey.Status.OwnerRef.Kind != "Operator" {
			logger.Info("signing key is not owned by an operator")
			return errors.NewBadRequest("signing key is not owned by an operator")
		}

		// TODO @JoeLanglands: ensure that this account can be managed by the operator that owns the signing key.
		// this involves following the namespace and label selector restrictions.

		// unpack the account claims from the spec
		accClaims := jwt.Account{
			Imports:    convertToNATSImports(acc.Spec.Imports),
			Exports:    convertToNATSExports(acc.Spec.Exports),
			Identities: convertToNATSIdentities(acc.Spec.Identities),
			Limits:     convertToNATSOperatorLimits(acc.Spec.Limits),
			// SigningKeys: []string{},  // TODO @JoeLanglands: add signing keys if they are needed, also what are they? I think they are the publicKeys of any signing keys belonging to the account.
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

		// now create the secrets for the jwt and seed TODO @JoeLanglands add labels and annotations
		jwtSecret := NewSecret(acc.Spec.JWTSecretName, acc.Namespace, WithData(map[string][]byte{"jwt": []byte(ajwt)}), WithImmutable(true))
		if _, err = r.CV1Interface.Secrets(acc.Namespace).Create(ctx, &jwtSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create jwt secret")
			return err
		}

		seedSecret := NewSecret(acc.Spec.SeedSecretName, acc.Namespace, WithData(map[string][]byte{"seed": seed}), WithImmutable(true))
		if _, err = r.CV1Interface.Secrets(acc.Namespace).Create(ctx, &seedSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create seed secret")
			return err
		}

		acc.Status.MarkJWTSecretReady()
		acc.Status.MarkSeedSecretReady(publicKey, seedSecret.Name)
	} else if errSeed != nil || errJWT != nil {
		err := multierr.Append(errSeed, errJWT)
		logger.Error(err, "failed to get seed or jwt secret")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&accountsnatsiov1alpha1.Account{}).
		Owns(&v1.Secret{}).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
