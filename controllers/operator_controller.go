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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
)

// OperatorReconciler reconciles a Operator object
type OperatorReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	CV1Interface      corev1.CoreV1Interface
	AccountsClientSet accountsclientsets.AccountsV1alpha1Interface
}

//+kubebuilder:rbac:groups=accounts.nats.io,resources=operators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=accounts.nats.io,resources=operators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=accounts.nats.io,resources=operators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *OperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("reconciling operator", "name", req.Name)

	operator := new(v1alpha1.Operator)
	if err := r.Get(ctx, req.NamespacedName, operator); err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("operator deleted")
			return ctrl.Result{}, nil
		}

		logger.Error(err, "failed to Get operator object")

		return ctrl.Result{}, err
	}

	originalStatus := operator.Status.DeepCopy()

	defer func() {
		if !equality.Semantic.DeepEqual(originalStatus, operator.Status) {
			if err2 := r.Status().Update(ctx, operator); err2 != nil {
				logger.Info("failed to update operator status", "error", err2.Error())

				err = multierr.Append(err, err2)
			}
		}
	}()

	if err := r.ensureSeedSecret(ctx, operator); err != nil {
		logger.Error(err, "failed to ensure seed secret")

		return ctrl.Result{}, err
	}

	sysAccId, err := r.ensureSystemAccountResolved(ctx, operator)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("system account not found or not ready, requeuing")
			return ctrl.Result{RequeueAfter: time.Second * 30}, nil
		}
		logger.Error(err, "failed to ensure system account resolved")

		return ctrl.Result{}, err
	}

	sKeys, err := r.ensureSigningKeysUpdated(ctx, operator)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("signing keys not found, requeuing")
			result.RequeueAfter = time.Second * 30
			return
		}
		logger.Error(err, "failed to ensure signing keys")

		return ctrl.Result{}, err
	}

	if err := r.ensureJWTSecret(ctx, operator, sKeys, sysAccId); err != nil {
		logger.Error(err, "failed to ensure JWT secret")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OperatorReconciler) ensureSeedSecret(ctx context.Context, operator *v1alpha1.Operator) error {
	logger := log.FromContext(ctx)

	// check if secret with operator seed exists
	var publicKey string
	secret, err := r.CV1Interface.Secrets(operator.Namespace).Get(ctx, operator.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		keyPair, err := nkeys.CreateOperator()
		if err != nil {
			logger.Error(err, "failed to create operator sk pair")
			return err
		}
		seed, err := keyPair.Seed()
		if err != nil {
			logger.Error(err, "failed to get operator seed")
			return err
		}
		publicKey, err = keyPair.PublicKey()
		if err != nil {
			logger.Error(err, "failed to get operator public sk")
			return err
		}

		labels := map[string]string{
			"operator-name": operator.Name,
			"secret-type":   string(v1alpha1.NatsSecretTypeSeed),
		}

		data := map[string][]byte{
			v1alpha1.NatsSecretSeedKey:      seed,
			v1alpha1.NatsSecretPublicKeyKey: []byte(publicKey),
		}

		seedSecret := NewSecret(operator.Spec.SeedSecretName, operator.Namespace, WithData(data), WithImmutable(true), WithLabels(labels))

		err = ctrl.SetControllerReference(operator, &seedSecret, r.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}

		secret, err = r.CV1Interface.Secrets(operator.Namespace).Create(ctx, &seedSecret, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create seed secret")
			return err
		}
	} else if err != nil {
		logger.Error(err, "failed to get seed secret")
		return err
	} else {
		publicKey = string(secret.Data[v1alpha1.NatsSecretPublicKeyKey])
	}

	operator.Status.MarkSeedSecretReady(publicKey, secret.Name)

	return nil
}

func (r *OperatorReconciler) ensureJWTSecret(ctx context.Context, operator *v1alpha1.Operator, sKeys []v1alpha1.SigningKeyEmbeddedStatus, sysAccId string) error {
	logger := log.FromContext(ctx)

	seedSecret, err := r.CV1Interface.Secrets(operator.Namespace).Get(ctx, operator.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		logger.V(1).Info("seed secret not found, skipping jwt secret creation")
		operator.Status.MarkJWTSecretFailed("seed secret not found", "")
		return nil
	}

	var sKeysPublicKeys []string
	for _, sk := range sKeys {
		sKeysPublicKeys = append(sKeysPublicKeys, sk.KeyPair.PublicKey)
	}

	operatorPublicKey := string(seedSecret.Data[v1alpha1.NatsSecretPublicKeyKey])

	jwtSec, err := r.CV1Interface.Secrets(operator.Namespace).Get(ctx, operator.Spec.JWTSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		op := jwt.Operator{
			SigningKeys:         sKeysPublicKeys,
			AccountServerURL:    operator.Spec.AccountServerURL,
			OperatorServiceURLs: operator.Spec.OperatorServiceURLs,
			SystemAccount:       sysAccId, // This should be resolved at this point
		}
		opClaims := jwt.NewOperatorClaims(operator.Status.KeyPair.PublicKey)
		opClaims.Name = operator.Name
		opClaims.Issuer = operatorPublicKey
		opClaims.IssuedAt = time.Now().Unix()
		opClaims.Type = jwt.OperatorClaim
		opClaims.Operator = op

		keys, err := nkeys.ParseDecoratedNKey(seedSecret.Data[v1alpha1.NatsSecretSeedKey])
		if err != nil {
			logger.Error(err, "failed to get nkeys from seed")
			return err
		}
		jwt, err := opClaims.Encode(keys)
		if err != nil {
			logger.Error(err, "failed to encode operator claims")
			return err
		}

		data := map[string][]byte{
			v1alpha1.NatsSecretJWTKey: []byte(jwt),
		}

		labels := map[string]string{
			"operator-name": operator.Name,
		}

		jwtSecret := NewSecret(operator.Spec.JWTSecretName, operator.Namespace, WithData(data), WithLabels(labels), WithImmutable(false))

		err = ctrl.SetControllerReference(operator, &jwtSecret, r.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}

		_, err = r.CV1Interface.Secrets(operator.Namespace).Create(ctx, &jwtSecret, metav1.CreateOptions{})
		if err != nil {
			operator.Status.MarkJWTSecretFailed("failed to create jwt secret for operator", "operator: %s", operator.Name)
			logger.Error(err, "failed to create jwt secret")
			return err
		}

	} else if err != nil {
		operator.Status.MarkJWTSecretFailed("could not find JWT secret for operator", "operator: %s", operator.Name)
		logger.Error(err, "failed to get jwt secret")
		return err
	} else {
		err := r.updateOperatorJWTSigningKeys(ctx, seedSecret.Data[v1alpha1.NatsSecretSeedKey], jwtSec, sKeysPublicKeys)
		if err != nil {
			logger.V(1).Info("failed to update operator JWT with signing keys", "error", err)
			operator.Status.MarkJWTSecretFailed("failed to update JWT with signing keys", "")
			return nil
		}
	}

	operator.Status.MarkJWTSecretReady()
	return nil
}

func (r *OperatorReconciler) ensureSigningKeysUpdated(ctx context.Context, operator *v1alpha1.Operator) ([]v1alpha1.SigningKeyEmbeddedStatus, error) {
	logger := log.FromContext(ctx)

	skList, err := r.AccountsClientSet.SigningKeys(operator.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil || skList == nil {
		logger.V(1).Info("failed to list signing keys", "error:", err)
		return nil, err
	}

	signingKeys := make([]v1alpha1.SigningKeyEmbeddedStatus, 0)
	for _, sk := range skList.Items {
		if sk.Status.IsReady() && sk.Status.OwnerRef.Namespace == operator.Namespace && sk.Status.OwnerRef.Name == operator.Name {
			signingKeys = append(signingKeys, v1alpha1.SigningKeyEmbeddedStatus{
				Name:    sk.GetName(),
				KeyPair: *sk.Status.KeyPair,
			})
		}
	}

	operator.Status.MarkSigningKeysUpdated(signingKeys)
	return signingKeys, nil
}

func (r *OperatorReconciler) ensureSystemAccountResolved(ctx context.Context, operator *v1alpha1.Operator) (string, error) {
	logger := log.FromContext(ctx)

	var sysAcc v1alpha1.Account
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: operator.Namespace,
		Name:      operator.Spec.SystemAccountRef.Name,
	}, &sysAcc); err != nil {
		operator.Status.MarkSystemAccountResolveFailed("system account not found", "account name: %s", operator.Spec.SystemAccountRef.Name)
		return "", err
	} else {
		operator.Status.MarkSystemAccountResolved(v1alpha1.InferredObjectReference{
			Namespace: sysAcc.GetNamespace(),
			Name:      sysAcc.GetName(),
		})
	}

	// TODO @JoeLanglands the system account should only be marked ready when it has a system user to be able to log in with!

	if !sysAcc.Status.IsReady() {
		logger.V(1).Info("system account not ready")
		operator.Status.MarkSystemAccountNotReady("system account not ready", "")
		return "", errors.NewNotFound(schema.GroupResource{
			Group:    v1alpha1.GroupVersion.Group,
			Resource: v1alpha1.Account{}.ResourceVersion,
		}, operator.Spec.SystemAccountRef.Name)
	} else {
		// The system account is ready, but does it have a system user to log in with?
		if err := r.ensureSystemAccountHasUser(ctx, &sysAcc); errors.IsNotFound(err) {
			logger.V(1).Info("system account has no system user")
			operator.Status.MarkSystemAccountNotReady("system account has no system user", "")
			return "", err
		} else if err != nil {
			logger.Error(err, "failed to ensure system account has system user")
			operator.Status.MarkSystemAccountNotReady("failed to ensure system account has system user", "")
			return "", err
		}
		// end clean-up
		operator.Status.MarkSystemAccountReady()
	}

	return sysAcc.Status.KeyPair.PublicKey, nil
}

func (r *OperatorReconciler) updateOperatorJWTSigningKeys(ctx context.Context, operatorSeed []byte, jwtSecret *v1.Secret, sKeys []string) error {
	logger := log.FromContext(ctx)

	operatorJWTEncoded := string(jwtSecret.Data[v1alpha1.NatsSecretJWTKey])
	opClaims, err := jwt.DecodeOperatorClaims(operatorJWTEncoded)
	if err != nil {
		logger.Error(err, "failed to decode operator jwt")
		return err
	}

	if isEqualUnordered(opClaims.SigningKeys, sKeys) {
		logger.V(1).Info("operator jwt signing keys are up to date")
		return nil
	}

	opClaims.SigningKeys = jwt.StringList(sKeys)

	kPair, err := nkeys.ParseDecoratedNKey(operatorSeed)
	if err != nil {
		logger.Error(err, "failed to get nkeys from seed")
		return err
	}
	ojwt, err := opClaims.Encode(kPair)
	if err != nil {
		logger.Error(err, "failed to encode operator jwt")
		return err
	}

	jwtSecret.Data[v1alpha1.NatsSecretJWTKey] = []byte(ojwt)
	_, err = r.CV1Interface.Secrets(jwtSecret.Namespace).Update(ctx, jwtSecret, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to update jwt secret")
		return err
	}

	return nil
}

func (r *OperatorReconciler) ensureSystemAccountHasUser(ctx context.Context, sysAcc *v1alpha1.Account) error {
	logger := log.FromContext(ctx)

	// All users of the system account should be within its namespace
	usrList, err := r.AccountsClientSet.Users(sysAcc.GetNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.V(1).Info("failed to list users for system account", "error", err)
		return err
	}

	for _, usr := range usrList.Items {
		if usr.Status.IsReady() && usr.Status.AccountRef.Name == sysAcc.GetName() {
			// System account has a user so return nil
			return nil
		}
	}

	return errors.NewNotFound(v1alpha1.Resource(v1alpha1.User{}.ResourceVersion), "User")
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("OperatorReconciler")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Operator{}).
		Owns(&v1.Secret{}).
		Watches(
			&source.Kind{Type: &v1alpha1.Account{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				// whenever an Account is created, updated or deleted, reconcile the Operator for which that account
				// belongs
				account, ok := obj.(*v1alpha1.Account)
				if !ok {
					logger.V(1).Info("Account watcher received non-Account object",
						"kind", obj.GetObjectKind().GroupVersionKind().String())

					return nil
				}

				operatorRef := account.Status.OperatorRef

				if operatorRef == nil {
					return nil
				}

				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name:      operatorRef.Name,
						Namespace: operatorRef.Namespace,
					},
				}}
			}),
		).
		Watches(
			&source.Kind{Type: &v1alpha1.SigningKey{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				// whenever a SigningKey is created, updated or deleted, check whether it's owner is an Operator, and
				// if so, reconcile it.
				signingKey, ok := obj.(*v1alpha1.SigningKey)
				if !ok {
					logger.V(1).Info("SigningKey watcher received non-SigningKey object",
						"kind", obj.GetObjectKind().GroupVersionKind().String())

					return nil
				}

				ownerRef := signingKey.Status.OwnerRef
				if ownerRef == nil {
					return nil
				}

				operatorGVK := v1alpha1.GroupVersion.WithKind("Operator")
				if operatorGVK != ownerRef.GetGroupVersionKind() {

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
}
