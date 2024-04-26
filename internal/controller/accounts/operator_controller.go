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
	"github.com/versori-oss/nats-account-operator/internal/controller/accounts/resources"
	"github.com/versori-oss/nats-account-operator/pkg/nsc"
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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStatus := operator.Status.DeepCopy()

	operator.Status.InitializeConditions()

	defer func() {
		if !equality.Semantic.DeepEqual(*originalStatus, operator.Status) {
			if err2 := r.Status().Update(ctx, operator); err2 != nil {
				if errors.IsConflict(err2) && err == nil {
					result = ctrl.Result{RequeueAfter: time.Second}

					return
				}

				err = multierr.Append(err, fmt.Errorf("failed to update operator status: %w", err2))
			}
		}
	}()

	kp, seed, result, err := r.reconcileSeedSecret(ctx, operator, nkeys.CreateOperator, operator.Spec.SeedSecretName,
		resources.Immutable(), resources.WithDeletionPrevention())
	if err != nil {
		MarkCondition(err, operator.Status.MarkSeedSecretFailed, operator.Status.MarkSeedSecretUnknown)

		return AsResult(err)
	}

	operator.Status.MarkSeedSecretReady(*kp)

	if !result.IsZero() {
		return result, nil
	}

	if err = r.ensureSystemAccountResolved(ctx, operator); err != nil {
		logger.Error(err, "failed to resolve system account")

		MarkCondition(err, operator.Status.MarkSystemAccountResolveFailed, operator.Status.MarkSystemAccountResolveUnknown)

		return AsResult(err)
	}

	if err = r.ensureSigningKeysUpdated(ctx, operator); err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("signing keys not found, requeuing")
			result.RequeueAfter = time.Second * 30

			return result, nil
		}

		logger.Error(err, "failed to ensure signing keys")

		return ctrl.Result{}, err
	}

	result, err = r.reconcileJWTSecret(ctx, operator, seed)
	if err != nil {
		MarkCondition(err, operator.Status.MarkJWTSecretFailed, operator.Status.MarkJWTSecretUnknown)

		return AsResult(err)
	}

	operator.Status.MarkJWTSecretReady()

	return result, nil
}

func (r *OperatorReconciler) reconcileJWTSecret(ctx context.Context, operator *v1alpha1.Operator, seed []byte) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	signingKey, err := nkeys.FromSeed(seed)
	if err != nil {
		return reconcile.Result{}, TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to get signing key from seed: %w", err))
	}

	// we want to check that any existing secret decodes to match wantClaims, if it doesn't then we will use nextJWT
	// to create/update the secret. We cannot just compare the JWTs from the secret and accountJWT because the JWTs are
	// timestamped with the `iat` claim so will never match.
	wantClaims, nextJWT, err := nsc.CreateOperatorClaims(operator, signingKey)
	if err != nil {
		return reconcile.Result{}, TerminalError(ConditionFailed(v1alpha1.ReasonUnknownError, "failed to create account JWT claims: %w", err))
	}

	got, err := r.CoreV1.Secrets(operator.Namespace).Get(ctx, operator.Spec.JWTSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("JWT secret not found, creating new secret")

			return reconcile.Result{Requeue: true}, r.createJWTSecret(ctx, operator, nextJWT)
		}

		return reconcile.Result{}, TemporaryError(ConditionUnknown(v1alpha1.ReasonUnknownError, "failed to get JWT secret: %w", err))
	}

	_, result, err := r.ensureJWTSecretUpToDate(ctx, operator, wantClaims, got, nextJWT)

	return result, err
}

func (r *OperatorReconciler) ensureSigningKeysUpdated(ctx context.Context, operator *v1alpha1.Operator) error {
	logger := log.FromContext(ctx)

	skList, err := r.AccountsV1Alpha1.SigningKeys(operator.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil || skList == nil {
		logger.V(1).Info("failed to list signing keys", "error:", err)

		return err
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

	return nil
}

// ensureSystemAccountResolved ensures we can resolve the system account for the operator, but doesn't require that the
// Account be ready.
//
// When setting up a new NATS deployment, we need both the system account public key, and the operator JWT to be
// written to the cluster. The associated Account resource will never become ready because it cannot push to the
// NATS deployment, so we update the operator's status correctly, but do not error out.
func (r *OperatorReconciler) ensureSystemAccountResolved(ctx context.Context, operator *v1alpha1.Operator) error {
	logger := log.FromContext(ctx)

	var sysAcc v1alpha1.Account
	if err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: operator.Namespace,
		Name:      operator.Spec.SystemAccountRef.Name,
	}, &sysAcc); err != nil {
		return ConditionFailed(v1alpha1.ReasonNotFound, "failed to get system account, %q: %w", operator.Spec.SystemAccountRef.Name, err)
	}

	if !sysAcc.GetConditionSet().Manage(&sysAcc.Status).GetCondition(v1alpha1.KeyPairableConditionSeedSecretReady).IsTrue() {
		logger.V(1).Info("system account does not have a KeyPair")

		return ConditionFailed(v1alpha1.ReasonNotReady, "system account KeyPair not ready")
	}

	operator.Status.MarkSystemAccountResolved(v1alpha1.KeyPairReference{
		InferredObjectReference: v1alpha1.InferredObjectReference{
			Namespace: sysAcc.GetNamespace(),
			Name:      sysAcc.GetName(),
		},
		PublicKey: sysAcc.Status.KeyPair.PublicKey,
	})

	return nil
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
	r.EventRecorder = mgr.GetEventRecorderFor("operator-controller")

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
