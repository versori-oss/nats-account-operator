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

	accSKey, err := r.ensureAccountResolved(ctx, usr)
	if err != nil {
		logger.Error(err, "failed to ensure owner resolved")
		return ctrl.Result{}, err
	}

	if err := r.ensureCredsSecrets(ctx, usr, accSKey); err != nil {
		logger.Error(err, "failed to ensure JWT seed secrets")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) ensureAccountResolved(ctx context.Context, usr *accountsnatsiov1alpha1.User) ([]byte, error) {
	logger := log.FromContext(ctx)

	sKey, err := r.AccountsClientSet.SigningKeys(usr.Namespace).Get(ctx, usr.Spec.SigningKey.Ref.Name, metav1.GetOptions{})
	if err != nil {
		usr.Status.MarkAccountResolveFailed("failed to get signing key for user", "")
		logger.Info("failed to get signing key for user", "user: %s", usr.Name)
		return []byte{}, err
	}

	skOwnerRef := sKey.Status.OwnerRef
	skOwnerRuntimeObj, _ := r.Scheme.New(skOwnerRef.GetGroupVersionKind())

	switch skOwnerRuntimeObj.(type) {
	case *accountsnatsiov1alpha1.Account:
		usr.Status.MarkAccountResolved(accountsnatsiov1alpha1.InferredObjectReference{
			Namespace: skOwnerRef.Namespace,
			Name:      skOwnerRef.Name,
		})
	default:
		usr.Status.MarkAccountResolveFailed("invalid signing key owner type", "signing key type: %s", skOwnerRef.Kind)
		return []byte{}, errors.NewBadRequest("invalid signing key owner type")
	}

	skSeedSecret, err := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, sKey.Spec.SeedSecretName, metav1.GetOptions{})
	if err != nil {
		logger.Info("failed to get account seed for signing key", "account: %s", usr.Status.AccountRef.Name, "signing key: %s", sKey.Name)
		return []byte{}, err
	}

	return skSeedSecret.Data[accountsnatsiov1alpha1.NatsSecretSeedKey], nil
}

func (r *UserReconciler) ensureCredsSecrets(ctx context.Context, usr *accountsnatsiov1alpha1.User, accSKey []byte) error {
	logger := log.FromContext(ctx)

	_, errSeed := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, usr.Spec.SeedSecretName, metav1.GetOptions{})
	_, errJWT := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, usr.Spec.JWTSecretName, metav1.GetOptions{})
	_, errCreds := r.CV1Interface.Secrets(usr.Namespace).Get(ctx, usr.Spec.CredentialsSecretName, metav1.GetOptions{})
	if errors.IsNotFound(errSeed) || errors.IsNotFound(errJWT) || errors.IsNotFound(errCreds) {
		// one or the other is not found, so re-create the creds, seed and jwt and then update/create the secrets

		kPair, err := nkeys.FromSeed(accSKey)
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

		userCreds, err := jwt.FormatUserConfig(ujwt, seed)
		if err != nil {
			logger.Info("failed to format user creds")
			usr.Status.MarkCredentialsSecretFailed("failed to format user creds", "")
			return nil
		}

		// create jwt secret and update or create it on the cluster
		jwtData := map[string][]byte{accountsnatsiov1alpha1.NatsSecretJWTKey: []byte(ujwt)}
		jwtSecret := NewSecret(usr.Spec.JWTSecretName, usr.Namespace, WithData(jwtData), WithImmutable(true))
		if err = ctrl.SetControllerReference(usr, &jwtSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set user as owner of jwt secret")
			return err
		}

		if err = r.createOrUpdateSecret(ctx, usr.Namespace, &jwtSecret, errJWT != nil); err != nil {
			logger.Error(err, "failed to create or update jwt secret")
			return err
		}

		// create seed secret and update or create it on the cluster
		seedData := map[string][]byte{
			accountsnatsiov1alpha1.NatsSecretSeedKey:      seed,
			accountsnatsiov1alpha1.NatsSecretPublicKeyKey: []byte(publicKey),
		}
		seedSecret := NewSecret(usr.Spec.SeedSecretName, usr.Namespace, WithData(seedData), WithImmutable(true))
		if err = ctrl.SetControllerReference(usr, &seedSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set user as owner of seed secret")
			return err
		}

		if err = r.createOrUpdateSecret(ctx, usr.Namespace, &seedSecret, errSeed != nil); err != nil {
			logger.Error(err, "failed to create or update seed secret")
			return err
		}

		// create creds secret and update or create it on the cluster
		credsData := map[string][]byte{accountsnatsiov1alpha1.NatsSecretCredsKey: userCreds}
		credsSecret := NewSecret(usr.Spec.CredentialsSecretName, usr.Namespace, WithData(credsData), WithImmutable(false))
		if err = ctrl.SetControllerReference(usr, &credsSecret, r.Scheme); err != nil {
			logger.Error(err, "failed to set user as owner of creds secret")
			return err
		}

		if err = r.createOrUpdateSecret(ctx, usr.Namespace, &credsSecret, errCreds != nil); err != nil {
			logger.Error(err, "failed to create or update creds secret")
			return err
		}

		usr.Status.MarkCredentialsSecretReady()
		usr.Status.MarkJWTSecretReady()
		usr.Status.MarkSeedSecretReady(publicKey, usr.Spec.SeedSecretName)
		return nil
	} else if errSeed != nil {
		// going to actually return and log errors here as something could have gone genuinely wrong
		logger.Error(errSeed, "failed to get seed secret")
		usr.Status.MarkSeedSecretUnknown("failed to get seed secret", "")
		return errSeed
	} else if errJWT != nil {
		logger.Error(errJWT, "failed to get jwt secret")
		usr.Status.MarkJWTSecretUnknown("failed to get jwt secret", "")
		return errJWT
	} else if errCreds != nil {
		logger.Error(errCreds, "failed to get credentials secret")
		usr.Status.MarkCredentialsSecretUnknown("failed to get credentials secrets", "")
		return errCreds
	}

	return nil
}

// createOrUpdateSecret will create or update a secret depending on the update flag. Pass true to update, false to create
// TODO @JoeLanglands there is a similar function in account_controller, should I just make these one function and pass a CV1Interface?
func (r *UserReconciler) createOrUpdateSecret(ctx context.Context, namespace string, secret *v1.Secret, update bool) error {
	var err error
	if update {
		_, err = r.CV1Interface.Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	} else {
		_, err = r.CV1Interface.Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	}

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&accountsnatsiov1alpha1.User{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
