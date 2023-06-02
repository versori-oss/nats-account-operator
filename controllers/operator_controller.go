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
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nkeys"
	accountsnatsiov1alpha1 "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
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
		if err != nil {
			return
		}
		if !equality.Semantic.DeepEqual(originalStatus, operator.Status) {
			// logger.Info("updating operator status", "status", operator.Status, "originalStatus", originalStatus)
			if err = r.Status().Update(ctx, operator); err != nil {
				if errors.IsConflict(err) {
					result.RequeueAfter = time.Second * 5
					return
				}
				logger.Error(err, "failed to update operator status")

			}
		}
	}()

	if err := r.ensureSeedSecret(ctx, operator); err != nil {
		logger.Error(err, "failed to ensure seed secret")

		return ctrl.Result{}, err
	}

	if err := r.ensureSigningKeys(ctx, operator); err != nil {
		logger.Error(err, "failed to ensure signing keys")

		return ctrl.Result{}, err
	}

	if err := r.ensureJWTSecret(ctx, operator); err != nil {
		logger.Error(err, "failed to ensure JWT secret")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OperatorReconciler) ensureSeedSecret(ctx context.Context, operator *accountsnatsiov1alpha1.Operator) error {
	logger := log.FromContext(ctx)

	// check if secret with operator seed exists
	var publicKey string
	secret, err := r.CV1Interface.Secrets(operator.Namespace).Get(ctx, operator.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		keyPair, err := nkeys.CreateOperator()
		if err != nil {
			logger.Error(err, "failed to create operator key pair")
			return err
		}
		seed, err := keyPair.Seed()
		if err != nil {
			logger.Error(err, "failed to get operator seed")
			return err
		}
		publicKey, err = keyPair.PublicKey()
		if err != nil {
			logger.Error(err, "failed to get operator public key")
			return err
		}

		labels := map[string]string{
			"operator-name": operator.Name,
			"secret-type":   string(accountsnatsiov1alpha1.NatsSecretTypeSeed),
		}

		data := map[string][]byte{
			"seed":      seed,
			"publicKey": []byte(publicKey),
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
		publicKey = string(secret.Data["publicKey"])
	}

	operator.Status.MarkSeedSecretReady(publicKey, secret.Name)

	return nil
}

func (r *OperatorReconciler) ensureJWTSecret(ctx context.Context, operator *accountsnatsiov1alpha1.Operator) error {
	logger := log.FromContext(ctx)

	seedSecret, err := r.CV1Interface.Secrets(operator.Namespace).Get(ctx, operator.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		logger.V(1).Info("seed secret not found, skipping jwt secret creation")
		return nil
	}

	operatorPublicKey := string(seedSecret.Data["publicKey"])

	_, err = r.CV1Interface.Secrets(operator.Namespace).Get(ctx, operator.Spec.JWTSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		opClaims := jwt.NewOperatorClaims(operator.Status.KeyPair.PublicKey)
		opClaims.Name = operator.Name
		opClaims.Issuer = operatorPublicKey
		opClaims.IssuedAt = time.Now().Unix()
		opClaims.Type = jwt.OperatorClaim
		keys, err := nkeys.FromSeed(seedSecret.Data["seed"])
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
			"jwt": []byte(jwt),
		}

		labels := map[string]string{
			"operator-name": operator.Name,
		}

		jwtSecret := NewSecret(operator.Spec.JWTSecretName, operator.Namespace, WithData(data), WithLabels(labels), WithImmutable(true))

		err = ctrl.SetControllerReference(operator, &jwtSecret, r.Scheme)
		if err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}

		_, err = r.CV1Interface.Secrets(operator.Namespace).Create(ctx, &jwtSecret, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create jwt secret")
			return err
		}
	} else if err != nil {
		logger.Error(err, "failed to get jwt secret")
		operator.Status.MarkJWTSecretFailed("Could not find JWT secret", "failed to get jwt secret: %s", err.Error())
		return err
	}

	operator.Status.MarkJWTSecretReady()

	return nil
}

func (r *OperatorReconciler) ensureSigningKeys(ctx context.Context, operator *accountsnatsiov1alpha1.Operator) error {
	logger := log.FromContext(ctx)

	sKeys, err := r.AccountsClientSet.SigningKeys(operator.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list signing keys")
		return err
	}

	// Need to filter by owner so that the ones iterated through are owned by operator

	var operatorSKeys []accountsnatsiov1alpha1.SigningKeyEmbeddedStatus

	for _, key := range sKeys.Items {
		// Dirty way because I'd like to use field selectors but they don't work so I may need to add labels later
		if key.Status.OwnerRef.Name == operator.Name {
			sKeyEmbedded := accountsnatsiov1alpha1.SigningKeyEmbeddedStatus{
				Name:    key.GetName(),
				KeyPair: *key.Status.KeyPair,
			}
			operatorSKeys = append(operatorSKeys, sKeyEmbedded)
		}
	}

	operator.Status.MarkSigningKeysUpdated(operatorSKeys)

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
