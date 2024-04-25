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

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"go.uber.org/multierr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

// OperatorReconciler reconciles a Operator object
type OperatorReconciler struct {
	*BaseReconciler
}

// +kubebuilder:rbac:groups=accounts.nats.io,resources=operators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=accounts.nats.io,resources=operators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=accounts.nats.io,resources=operators/finalizers,verbs=update

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

	operator.Status.InitializeConditions()

	defer func() {
		if !equality.Semantic.DeepEqual(*originalStatus, operator.Status) {
			if err2 := r.Status().Update(ctx, operator); err2 != nil {
				logger.Info("failed to update operator status", "error", err2.Error())

				err = multierr.Append(err, err2)
			}
		}
	}()

	kp, _, err := r.reconcileSeedSecret(ctx, operator, nkeys.CreateOperator, operator.Spec.SeedSecretName)
	if err != nil {
		MarkCondition(err, operator.Status.MarkSeedSecretFailed, operator.Status.MarkSeedSecretUnknown)

		return AsResult(err)
	}

	operator.Status.MarkSeedSecretReady(*kp)

	sysAccId, err := r.ensureSystemAccountResolved(ctx, operator)
	if err != nil {
		logger.Error(err, "failed to resolve system account")

		MarkCondition(err, operator.Status.MarkSystemAccountResolveFailed, operator.Status.MarkSystemAccountResolveUnknown)

		return AsResult(err)
	}

	sKeys, err := r.ensureSigningKeysUpdated(ctx, operator)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("signing keys not found, requeuing")
			result.RequeueAfter = time.Second * 30

			return result, nil
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

func (r *OperatorReconciler) ensureJWTSecret(ctx context.Context, operator *v1alpha1.Operator, sKeys []v1alpha1.SigningKeyEmbeddedStatus, sysAccId string) error {
	logger := log.FromContext(ctx)

	seedSecret, err := r.CoreV1.Secrets(operator.Namespace).Get(ctx, operator.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		logger.V(1).Info("seed secret not found, skipping jwt secret creation")
		operator.Status.MarkJWTSecretFailed("seed secret not found", "")

		return nil
	}

	sKeysPublicKeys := make([]string, len(sKeys))
	for i, sk := range sKeys {
		sKeysPublicKeys[i] = sk.KeyPair.PublicKey
	}

	operatorPublicKey := string(seedSecret.Data[v1alpha1.NatsSecretPublicKeyKey])

	jwtSec, err := r.CoreV1.Secrets(operator.Namespace).Get(ctx, operator.Spec.JWTSecretName, metav1.GetOptions{})
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

		_, err = r.CoreV1.Secrets(operator.Namespace).Create(ctx, &jwtSecret, metav1.CreateOptions{})
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

	skList, err := r.AccountsV1Alpha1.SigningKeys(operator.Namespace).List(ctx, metav1.ListOptions{})
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

// ensureSystemAccountResolved ensures we can resolve the system account for the operator, but doesn't require that the
// Account be ready.
//
// When setting up a new NATS deployment, we need both the system account public key, and the operator JWT to be
// written to the cluster. The associated Account resource will never become ready because it cannot push to the
// NATS deployment, so we update the operator's status correctly, but do not error out.
func (r *OperatorReconciler) ensureSystemAccountResolved(ctx context.Context, operator *v1alpha1.Operator) (string, error) {
	logger := log.FromContext(ctx)

	var sysAcc v1alpha1.Account
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: operator.Namespace,
		Name:      operator.Spec.SystemAccountRef.Name,
	}, &sysAcc); err != nil {
		return "", ConditionFailed(v1alpha1.ReasonNotFound, "failed to get system account, %q: %w", operator.Spec.SystemAccountRef.Name, err)
	}

	if !sysAcc.GetConditionSet().Manage(&sysAcc.Status).GetCondition(v1alpha1.KeyPairableConditionSeedSecretReady).IsTrue() {
		logger.V(1).Info("system account does not have a KeyPair")

		return "", ConditionFailed(v1alpha1.ReasonNotReady, "system account KeyPair not ready")
	}

	operator.Status.MarkSystemAccountResolved(v1alpha1.InferredObjectReference{
		Namespace: sysAcc.GetNamespace(),
		Name:      sysAcc.GetName(),
	})

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
	_, err = r.CoreV1.Secrets(jwtSecret.Namespace).Update(ctx, jwtSecret, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to update jwt secret")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("OperatorReconciler")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Operator{}).
		Owns(&v1.Secret{}).
		Watches(
			&v1alpha1.Account{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
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
			&v1alpha1.SigningKey{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
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
