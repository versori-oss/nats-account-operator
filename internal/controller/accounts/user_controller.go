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
	"time"

	goerrors "github.com/go-faster/errors"
	"github.com/go-logr/logr"
	"github.com/nats-io/nkeys"
	"go.uber.org/multierr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	"github.com/versori-oss/nats-account-operator/internal/controller/accounts/resources"
	"github.com/versori-oss/nats-account-operator/pkg/helpers"
	"github.com/versori-oss/nats-account-operator/pkg/nsc"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	*BaseReconciler
}

//+kubebuilder:rbac:groups=accounts.nats.io,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=accounts.nats.io,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=accounts.nats.io,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	usr := new(v1alpha1.User)
	if err := r.Client.Get(ctx, req.NamespacedName, usr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStatus := usr.Status.DeepCopy()

	usr.Status.InitializeConditions()

	defer func() {
		if !equality.Semantic.DeepEqual(*originalStatus, usr.Status) {
			if err2 := r.Status().Update(ctx, usr); err2 != nil {
				if errors.IsConflict(err2) && err == nil {
					result = ctrl.Result{RequeueAfter: time.Second}

					return
				}

				logger.Info("failed to update user status", "error", err2.Error())

				err = multierr.Append(err, err2)
			}
		}
	}()

	kp, seed, result, err := r.reconcileSeedSecret(ctx, usr, nkeys.CreateUser, usr.Spec.SeedSecretName)
	if err != nil {
		logger.Error(err, "failed to reconcile seed secret")

		MarkCondition(err, usr.Status.MarkSeedSecretFailed, usr.Status.MarkSeedSecretUnknown)

		return AsResult(err)
	}

	usr.Status.MarkSeedSecretReady(*kp)

	if !result.IsZero() {
		return result, nil
	}

	// get the KeyPairable which will be used to sign the JWT, resolveIssuer is part of BaseReconciler which doesn't
	// mark conditions (since it doesn't know what resource type it's reconciling), so we need to check for condition
	// errors and mark the conditions accordingly
	keyPairable, err := r.resolveIssuer(ctx, usr.Spec.Issuer, usr.Namespace)
	if err != nil {
		MarkCondition(err, usr.Status.MarkIssuerResolveFailed, usr.Status.MarkIssuerResolveUnknown)

		return AsResult(err)
	}

	acc, err := r.resolveAccount(ctx, usr, keyPairable)
	if err != nil {
		return AsResult(err)
	}

	if err := r.validateAccountSelector(ctx, acc, usr); err != nil {
		MarkCondition(err, usr.Status.MarkAccountResolveFailed, usr.Status.MarkAccountResolveUnknown)

		return AsResult(err)
	}

	usr.Status.MarkAccountResolved(v1alpha1.InferredObjectReference{
		Namespace: acc.Namespace,
		Name:      acc.Name,
	})

	result, err = r.reconcileLabels(ctx, usr)
	if !result.IsZero() || err != nil {
		return result, err
	}

	logger.V(1).Info("reconciling user JWT secret")

	ujwt, result, err := r.reconcileJWTSecret(ctx, usr, acc, keyPairable)
	if err != nil {
		MarkCondition(err, usr.Status.MarkJWTSecretFailed, usr.Status.MarkJWTSecretUnknown)

		return AsResult(err)
	}

	usr.Status.MarkJWTSecretReady()

	if !result.IsZero() {
		return result, nil
	}

	logger.V(1).Info("reconciling user credential secret")

	result, err = r.reconcileUserCredentialSecret(ctx, usr, acc, ujwt, seed)
	if err != nil {
		return result, fmt.Errorf("failed to reconcile user credential secret: %w", err)
	}

	return result, nil
}


func (r *UserReconciler) validateAccountSelector(ctx context.Context, account *v1alpha1.Account, user *v1alpha1.User) error {
	ns, err := r.CoreV1.Namespaces().Get(ctx, user.Namespace, metav1.GetOptions{})
	if err != nil {
		return TemporaryError(fmt.Errorf("failed to get user namespace: %w", err))
	}

	valid, err := helpers.MatchNamespaceSelector(account, ns, account.Spec.UsersNamespaceSelector)
	if err != nil {
		return TerminalError(fmt.Errorf("failed to validate users namespace selector: %w", err))
	}

	if !valid {
		return TerminalError(ConditionFailed(v1alpha1.ReasonNotAllowed, "account.spec.usersNamespaceSelector does not match user namespace"))
	}

	if account.Spec.UsersSelector == nil {
		// nothing else to validate, user is allowed.
		return nil
	}

	ls, err := metav1.LabelSelectorAsSelector(account.Spec.UsersSelector)
	if err != nil {
		return TerminalError(fmt.Errorf("failed to parse account.spec.usersSelector: %w", err))
	}

	if !ls.Matches(labels.Set(user.Labels)) {
		return TerminalError(ConditionFailed(v1alpha1.ReasonNotAllowed, "account.spec.usersSelector does not match user labels"))
	}

	return nil
}

// resolveAccount handles the v1alpha1.UserConditionAccountResolved condition and updating the
// .status.operatorRef field. If the provided keyPair is a SigningKey this will correctly resolve the owner to an
// Operator.
func (r *UserReconciler) resolveAccount(ctx context.Context, user *v1alpha1.User, keyPair v1alpha1.KeyPairable) (account *v1alpha1.Account, err error) {
	logger := log.FromContext(ctx)

	switch v := keyPair.(type) {
	case *v1alpha1.Account:
		logger.V(1).Info("user issuer is an account")

		account = v
	case *v1alpha1.SigningKey:
		logger.V(1).Info("user issuer is a signing key, resolving owner")

		owner, err := r.resolveSigningKeyOwner(ctx, v)
		if err != nil {
			MarkCondition(err, user.Status.MarkAccountResolveFailed, user.Status.MarkAccountResolveUnknown)

			return nil, err
		}

		var ok bool

		if account, ok = owner.(*v1alpha1.Account); !ok {
			user.Status.MarkAccountResolveFailed(v1alpha1.ReasonInvalidSigningKeyOwner, "user issuer is not owned by an Account, got: %s", owner.GetObjectKind().GroupVersionKind().String())

			return nil, TerminalError(fmt.Errorf("user issuer is not owned by an Account"))
		}

		// ensure that account is "Ready", since we're going to assume public key etc., is defined.
		conditions := account.GetConditionSet().Manage(account.GetStatus())

		// Initialize the conditions if they are not already set, not doing this causes a nil-pointer dereference panic
		conditions.InitializeConditions()

		accountReadyCondition := conditions.GetCondition(v1alpha1.AccountConditionReady)
		if !accountReadyCondition.IsTrue() {
			logger.V(1).Info("signing key owner is not ready", "reason", accountReadyCondition.Reason, "message", accountReadyCondition.Message)

			return nil, TemporaryError(ConditionUnknown(
				v1alpha1.ReasonNotReady, "signing key owner is not ready"))
		}
	default:
		logger.Info("invalid keypair, expected Account or SigningKey", "key_pair_type", fmt.Sprintf("%T", keyPair))

		user.Status.MarkAccountResolveFailed(v1alpha1.ReasonUnsupportedIssuer, "invalid keypair, expected Account or SigningKey, got: %s", keyPair.GroupVersionKind().String())

		return nil, TerminalError(fmt.Errorf("invalid keypair, expected Account or SigningKey"))
	}

	return account, nil
}

func (r *UserReconciler) reconcileJWTSecret(ctx context.Context, usr *v1alpha1.User, account *v1alpha1.Account, keyPairable v1alpha1.KeyPairable) (string, reconcile.Result, error) {
	logger := log.FromContext(ctx)

	issuerKP, err := r.loadIssuerSeed(ctx, keyPairable, nkeys.PrefixByteAccount)
	if err != nil {
		MarkCondition(err, usr.Status.MarkIssuerResolveFailed, usr.Status.MarkIssuerResolveUnknown)

		return "", reconcile.Result{}, err
	}

	usr.Status.MarkIssuerResolved()

	// we want to check that any existing secret decodes to match wantClaims, if it doesn't then we will use nextJWT
	// to create/update the secret. We cannot just compare the JWTs from the secret and accountJWT because the JWTs are
	// timestamped with the `iat` claim so will never match.
	wantClaims, nextJWT, err := nsc.CreateUserClaims(usr, account, issuerKP)
	if err != nil {
		return "", reconcile.Result{}, TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to create user JWT claims: %w", err))
	}

	got, err := r.CoreV1.Secrets(usr.Namespace).Get(ctx, usr.Spec.JWTSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("JWT secret not found, creating new secret")

			return nextJWT, reconcile.Result{Requeue: true}, r.createJWTSecret(ctx, usr, nextJWT)
		}

		return "", reconcile.Result{}, TemporaryError(ConditionUnknown(v1alpha1.ReasonUnknownError, "failed to get JWT secret: %w", err))
	}

	return r.ensureJWTSecretUpToDate(ctx, usr, wantClaims, got, nextJWT)
}

func (r *UserReconciler) getCAIfExists(ctx context.Context, acc *v1alpha1.Account) ([]byte, error) {
	logger := log.FromContext(ctx)

	if acc.Status.OperatorRef == nil {
		logger.Info("accounts has nil operator ref")
		return nil, errInternalNotFound
	}

	operatorRef := acc.Status.OperatorRef

	opClient := r.AccountsV1Alpha1.Operators(operatorRef.Namespace)
	op, err := opClient.Get(ctx, operatorRef.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "failed to retrieve the operator for account")
		return nil, err
	}

	if op.Spec.TLSConfig == nil {
		return nil, errInternalNotFound
	}

	s, err := r.CoreV1.Secrets(op.Namespace).Get(ctx, op.Spec.TLSConfig.CAFile.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "could not find secret")
			return nil, errInternalNotFound
		}
		return nil, err
	}

	caData, ok := s.Data[op.Spec.TLSConfig.CAFile.Key]
	if !ok {
		logger.Info("CA key not found in secret", "secret", fmt.Sprintf("%s/%s", acc.Spec.Issuer.Ref.Namespace, op.Spec.TLSConfig.CAFile.Name))
		return nil, errInternalNotFound
	}

	return caData, nil
}

func (r *UserReconciler) reconcileUserCredentialSecret(ctx context.Context, usr *v1alpha1.User, acc *v1alpha1.Account, ujwt string, seed []byte) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	ca, err := r.getCAIfExists(ctx, acc)
	if err != nil {
		if !goerrors.Is(err, errInternalNotFound) {
			return reconcile.Result{}, err
		}

		logger.V(1).Info("no TLS config found for account operator")
	}

	got, err := r.CoreV1.Secrets(usr.Namespace).Get(ctx, usr.Spec.CredentialsSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("credentials secret not found, creating new secret")

			if err := r.createCredentialsSecret(ctx, usr, ujwt, seed, ca); err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{Requeue: true}, nil
		}

		logger.Error(err, "failed to get credentials secret")

		usr.Status.MarkCredentialsSecretUnknown(v1alpha1.ReasonUnknownError, err.Error())

		return reconcile.Result{}, err
	}

	return r.ensureCredentialsSecretUpToDate(ctx, usr, ujwt, seed, got, ca)
}

func (r *UserReconciler) createCredentialsSecret(ctx context.Context, usr *v1alpha1.User, ujwt string, seed []byte, ca []byte) error {
	logger := log.FromContext(ctx)

	secret, err := resources.NewUserCredentialSecretBuilder(r.Scheme, ca).Build(usr, ujwt, seed)
	if err != nil {
		logger.Error(err, "failed to build credentials secret")

		usr.Status.MarkCredentialsSecretFailed(v1alpha1.ReasonUnknownError, err.Error())

		return err
	}

	if err := r.Client.Create(ctx, secret); err != nil {
		logger.Error(err, "failed to create credentials secret")

		usr.Status.MarkCredentialsSecretFailed(v1alpha1.ReasonUnknownError, err.Error())

		return err
	}

	usr.Status.MarkCredentialsSecretReady()

	r.EventRecorder.Eventf(usr, v1.EventTypeNormal, "CredentialsSecretCreated", "created secret: %s/%s", secret.Namespace, secret.Name)

	return nil
}

func (r *UserReconciler) ensureCredentialsSecretUpToDate(ctx context.Context, usr *v1alpha1.User, ujwt string, seed []byte, got *v1.Secret, ca []byte) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	want, err := resources.NewUserCredentialSecretBuilderFromSecret(got.DeepCopy(), r.Scheme, ca).Build(usr, ujwt, seed)
	if err != nil {
		err = fmt.Errorf("failed to build desired credentials secret: %w", err)

		usr.Status.MarkCredentialsSecretFailed(v1alpha1.ReasonUnknownError, err.Error())

		return reconcile.Result{}, err
	}

	if equality.Semantic.DeepEqual(got, want) {
		logger.V(5).Info("existing credentials secret matches desired state, no update required")

		usr.Status.MarkCredentialsSecretReady()

		return reconcile.Result{}, nil
	}

	if err := r.Update(ctx, want); err != nil {
		err = fmt.Errorf("failed to update credentials secret: %w", err)

		usr.Status.MarkCredentialsSecretFailed(v1alpha1.ReasonUnknownError, err.Error())

		return reconcile.Result{}, err
	}

	r.EventRecorder.Eventf(usr, v1.EventTypeNormal, "CredentialsSecretUpdated", "updated secret: %s/%s", want.Namespace, want.Name)

	usr.Status.MarkCredentialsSecretReady()

	return reconcile.Result{Requeue: true}, nil
}

func (r *UserReconciler) reconcileLabels(ctx context.Context, user *v1alpha1.User) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	if user.Status.AccountRef == nil {
		return reconcile.Result{}, nil
	}

	if user.Labels == nil {
		user.Labels = make(map[string]string)
	}

	if user.Labels[resources.LabelAccountName] != user.Status.AccountRef.Name {
		user.Labels[resources.LabelAccountName] = user.Status.AccountRef.Name

		if err := r.Update(ctx, user); err != nil {
			logger.Error(err, "failed to update user labels")

			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("user-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.User{}).
		Owns(&v1.Secret{}).
		Watches(&v1alpha1.Account{}, userAccountWatcher(mgr.GetLogger(), mgr.GetClient())).
		Complete(r)
}

func userAccountWatcher(logger logr.Logger, c client.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		account, ok := obj.(*v1alpha1.Account)
		if !ok {
			logger.Info("Account watcher received non-Account object",
				"kind", obj.GetObjectKind().GroupVersionKind().String())
			return nil
		}

		var users v1alpha1.UserList

		if err := c.List(ctx, &users, client.MatchingLabels{
			resources.LabelAccountName: account.Name,
		}); err != nil {
			logger.Error(err, "failed to list users for account", "account", account.Name)

			return nil
		}

		requests := make([]reconcile.Request, len(users.Items))

		for i, user := range users.Items {
			requests[i] = reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(&user),
			}
		}

		return requests
	})
}
