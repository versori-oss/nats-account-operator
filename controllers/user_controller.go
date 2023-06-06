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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nkeys"
	accountsnatsiov1alpha1 "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	CV1Interface      corev1.CoreV1Interface
	AccountsClientSet accountsclientsets.AccountsV1alpha1Interface
}

//+kubebuilder:rbac:groups=accounts.nats.io,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=accounts.nats.io,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=accounts.nats.io,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	usr := new(accountsnatsiov1alpha1.User)
	if err := r.Client.Get(ctx, req.NamespacedName, usr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("user deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to fetch user")
		return ctrl.Result{}, err
	}

	originalStatus := usr.Status.DeepCopy()

	defer func() {
		if err != nil {
			return
		}
		if !equality.Semantic.DeepEqual(originalStatus, usr.Status) {
			if err = r.Status().Update(ctx, usr); err != nil {
				if errors.IsConflict(err) {
					result.RequeueAfter = time.Second * 5
					return
				}
				logger.Error(err, "failed to update user status")
			}
		}
	}()

	if err := r.ensureCredsSecrets(ctx, usr); err != nil {
		logger.Error(err, "failed to ensure JWT seed secrets")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) ensureCredsSecrets(ctx context.Context, usr *accountsnatsiov1alpha1.User) error {
	logger := log.FromContext(ctx)

	_, errSeed := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, usr.Spec.SeedSecretName, metav1.GetOptions{})
	_, errJWT := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, usr.Spec.JWTSecretName, metav1.GetOptions{})
	if errors.IsNotFound(errSeed) || errors.IsNotFound(errJWT) {
		sKey, err := r.AccountsClientSet.SigningKeys(usr.Namespace).Get(ctx, usr.Spec.SigningKey.Ref.Name, metav1.GetOptions{})
		if err != nil {
			logger.Info("signing key not found")
			usr.Status.MarkAccountResolveFailed("signing key not found", "%s:%s", usr.Spec.SigningKey.Ref.Namespace, usr.Spec.SigningKey.Ref.Name)
			return nil
		}
		if sKey.Status.OwnerRef.Kind != "Account" {
			logger.Info("signing key not owned by account")
			usr.Status.MarkAccountResolveFailed("signing key not owned by an account", "%s:%s", usr.Spec.SigningKey.Ref.Namespace, usr.Spec.SigningKey.Ref.Name)
			return nil
		}

		skSeedSecret, err := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, sKey.Spec.SeedSecretName, metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "failed to get signing key seed secret")
			return err
		}

		kPair, err := nkeys.FromSeed(skSeedSecret.Data["seed"])
		if err != nil {
			logger.Error(err, "failed to make key pair from seed")
			return err
		}

		usrClaims := jwt.User{
			Permissions: convertToNATSUserPermissions(usr.Spec.Permissions),
			Limits:      convertToNATSLimits(usr.Spec.Limits),
			BearerToken: usr.Spec.BearerToken,
		}

		ujwt, publicKey, seed, err := CreateUser(usr.Name, usrClaims, kPair)
		if err != nil {
			logger.Error(err, "failed to create user jwt")
			return err
		}

		// now create the secrets for the jkt and seed TODO @JoeLanglands add labels and annotations
		jwtSecret := NewSecret(usr.Spec.JWTSecretName, usr.Namespace, WithData(map[string][]byte{"jwt": []byte(ujwt)}), WithImmutable(true))
		if _, err = r.CV1Interface.Secrets(usr.Namespace).Create(ctx, &jwtSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create jwt secret")
			return err
		}

		if err = ctrl.SetControllerReference(usr, &jwtSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set user as owner of jwt secret")
			return err
		}

		seedSecret := NewSecret(usr.Spec.SeedSecretName, usr.Namespace, WithData(map[string][]byte{"seed": seed}), WithImmutable(true))
		if _, err := r.CV1Interface.Secrets(usr.Namespace).Create(ctx, &seedSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create seed secret")
			return err
		}

		if err = ctrl.SetControllerReference(usr, &seedSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set user as owner of seed secret")
			return err
		}

		userCreds, err := jwt.FormatUserConfig(ujwt, seed)
		if err != nil {
			logger.Info("failed to format user creds")
			usr.Status.MarkCredentialsSecretFailed("failed to format user creds", "")
			return nil
		}
		credsSecret := NewSecret(usr.Spec.CredentialsSecretName, usr.Namespace, WithData(map[string][]byte{"creds": []byte(userCreds)}), WithImmutable(true))
		if _, err := r.CV1Interface.Secrets(usr.Namespace).Create(ctx, &credsSecret, metav1.CreateOptions{}); err != nil {
			logger.Error(err, "failed to create creds secret")
			return err
		}

		if err = ctrl.SetControllerReference(usr, &credsSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set user as owner of creds secret")
			return err
		}

		usr.Status.MarkCredentialsSecretReady()
		usr.Status.MarkJWTSecretReady()
		usr.Status.MarkSeedSecretReady(publicKey, usr.Spec.SeedSecretName)
	} else if errSeed != nil || errJWT != nil {
		// TODO @JoeLanglands change this shit
		err := multierr.Append(errSeed, errJWT)
		logger.Error(err, "failed to get jwt or seed secret")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&accountsnatsiov1alpha1.User{}).
		Complete(r)
}
