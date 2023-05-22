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
	"go.uber.org/multierr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	accountsnatsiov1alpha1 "github.com/versori-oss/nats-account-operator/api/v1alpha1"
)

// OperatorReconciler reconciles a Operator object
type OperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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

	operator := new(accountsnatsiov1alpha1.Operator)
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
				logger.Error(err, "failed to update operator status")

				err = multierr.Append(err, err2)
			}
		}
	}()

	if err := r.ensureSeedSecret(ctx, operator); err != nil {
		logger.Error(err, "failed to ensure seed secret")

		return ctrl.Result{}, err
	}

	if err := r.ensureJWTSecret(ctx, operator); err != nil {
		logger.Error(err, "failed to ensure JWT secret")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OperatorReconciler) ensureSeedSecret(ctx context.Context, operator *accountsnatsiov1alpha1.Operator) error {
	//seedSecret := new(v1.Secret)
	//seedName := types.NamespacedName{
	//    Name:      operator.Spec.SeedSecretName,
	//    Namespace: operator.Namespace,
	//}

	operator.Status.MarkSeedSecretReady("", "")

	return nil
}

func (r *OperatorReconciler) ensureJWTSecret(ctx context.Context, operator *accountsnatsiov1alpha1.Operator) error {
	operator.Status.MarkJWTSecretReady()

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger().WithName("OperatorReconciler")

	return ctrl.NewControllerManagedBy(mgr).
		For(&accountsnatsiov1alpha1.Operator{}).
		Owns(&v1.Secret{}).
		Watches(
			&source.Kind{Type: &accountsnatsiov1alpha1.Account{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				// whenever an Account is created, updated or deleted, reconcile the Operator for which that account
				// belongs
				account, ok := obj.(*accountsnatsiov1alpha1.Account)
				if !ok {
					logger.Info("Account watcher received non-Account object",
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
			&source.Kind{Type: &accountsnatsiov1alpha1.SigningKey{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				// whenever a SigningKey is created, updated or deleted, check whether it's owner is an Operator, and
				// if so, reconcile it.
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

				operatorGVK := (&accountsnatsiov1alpha1.Operator{}).GetObjectKind().GroupVersionKind()
				if operatorGVK != ownerRef.GetGroupVersionKind() {
					// TODO: remove this log once we're happy the != is handled correctly
					logger.V(1).Info("SigningKey watcher received SigningKey with non-operator owner",
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
}
