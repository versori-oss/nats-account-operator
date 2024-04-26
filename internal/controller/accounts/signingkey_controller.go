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

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nats-io/nkeys"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	"github.com/versori-oss/nats-account-operator/internal/controller/accounts/resources"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
)

// SigningKeyReconciler reconciles a SigningKey object
type SigningKeyReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	CoreV1           corev1.CoreV1Interface
	AccountsV1Alpha1 accountsclientsets.AccountsV1alpha1Interface
}

//+kubebuilder:rbac:groups=accounts.nats.io,resources=signingkeys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=accounts.nats.io,resources=signingkeys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=accounts.nats.io,resources=signingkeys/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SigningKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("reconciling signing key", "name", req.Name)

	signingKey := new(v1alpha1.SigningKey)
	if err := r.Get(ctx, req.NamespacedName, signingKey); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStatus := signingKey.Status.DeepCopy()

	signingKey.Status.InitializeConditions()

	defer func() {
		if !equality.Semantic.DeepEqual(*originalStatus, signingKey.Status) {
			if err2 := r.Status().Update(ctx, signingKey); err2 != nil {
				if errors.IsConflict(err2) && err == nil {
					result = ctrl.Result{RequeueAfter: time.Second}

					return
				}

				err = multierr.Append(err, fmt.Errorf("failed to update signing key status: %w", err2))
			}
		}
	}()

	if err := r.ensureOwnerResolved(ctx, signingKey); err != nil {
		signingKey.Status.MarkOwnerResolveFailed(v1alpha1.ReasonUnknownError, "failed to resolve owner: %s", err.Error())

		return AsResult(err)
	}

	result, err = r.reconcileLabels(ctx, signingKey)
	if !result.IsZero() || err != nil {
		return result, err
	}

	result, err = r.ensureKeyPair(ctx, signingKey)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure key pair: %w", err)
	}

	return result, nil
}

func (r *SigningKeyReconciler) ensureKeyPair(ctx context.Context, signingKey *v1alpha1.SigningKey) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	var publicKey string
	secret, err := r.CoreV1.Secrets(signingKey.Namespace).Get(ctx, signingKey.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		var keyPair nkeys.KeyPair
		switch signingKey.Spec.OwnerRef.Kind {
		case v1alpha1.SigningKeyTypeAccount:
			keyPair, err = nkeys.CreateAccount()
		case v1alpha1.SigningKeyTypeOperator:
			keyPair, err = nkeys.CreateOperator()
		default:
			err := errors.NewBadRequest(fmt.Sprintf("unknown owner kind: %s", signingKey.Spec.OwnerRef.Kind))
			return reconcile.Result{}, err
		}
		if err != nil {
			logger.Error(err, "failed to create key pair")
			return reconcile.Result{}, err
		}

		seed, err := keyPair.Seed()
		if err != nil {
			logger.Error(err, "failed to get seed")
			return reconcile.Result{}, err
		}
		publicKey, err = keyPair.PublicKey()
		if err != nil {
			logger.Error(err, "failed to get public key")
			return reconcile.Result{}, err
		}

		data := map[string][]byte{
			v1alpha1.NatsSecretSeedKey:      seed,
			v1alpha1.NatsSecretPublicKeyKey: []byte(publicKey),
		}

		labels := map[string]string{
			"operator-name": signingKey.Spec.OwnerRef.Name,
			"secret-type":   string(v1alpha1.NatsSecretTypeSKey),
		}

		secret := NewSecret(signingKey.Spec.SeedSecretName, signingKey.Namespace, WithImmutable(true), WithLabels(labels), WithData(data))

		if err = ctrl.SetControllerReference(signingKey, &secret, r.Scheme); err != nil {
			logger.Error(err, "failed to set controller reference")
			return reconcile.Result{}, err
		}

		_, err = r.CoreV1.Secrets(signingKey.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create seed secret")
			return reconcile.Result{}, err
		}

		signingKey.Status.MarkSeedSecretReady(publicKey, signingKey.Spec.SeedSecretName)

		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "failed to fetch seed secret")
		return reconcile.Result{}, err
	} else {
		publicKey = string(secret.Data[v1alpha1.NatsSecretPublicKeyKey])
	}

	signingKey.Status.MarkSeedSecretReady(publicKey, signingKey.Spec.SeedSecretName)

	return reconcile.Result{}, nil
}

func (r *SigningKeyReconciler) ensureOwnerResolved(ctx context.Context, signingKey *v1alpha1.SigningKey) error {
	logger := log.FromContext(ctx)

	ownerRef := signingKey.Spec.OwnerRef
	ownerGVK := schema.FromAPIVersionAndKind(ownerRef.APIVersion, ownerRef.Kind)

	ownerRuntimeObj, _ := r.Scheme.New(ownerGVK)
	switch ownerRuntimeObj.(type) {
	case *v1alpha1.Account, *v1alpha1.Operator:
		break
	default:
		signingKey.Status.MarkOwnerResolveFailed("UnsupportedOwnerKind", "owner must be one of Account or Operator")

		return nil
	}

	ownerObj := ownerRuntimeObj.(client.Object)

	if err := r.Client.Get(ctx, types.NamespacedName{
		// SigningKey owners must be in the same namespace as the SigningKey
		Namespace: signingKey.Namespace,
		Name:      ownerRef.Name,
	}, ownerObj); err != nil {
		if errors.IsNotFound(err) {
			signingKey.Status.MarkOwnerResolveFailed(v1alpha1.ReasonNotFound, "%s, %s/%s: not found", ownerGVK, signingKey.Namespace, ownerRef.Name)

			return err
		}

		signingKey.Status.MarkOwnerResolveUnknown(v1alpha1.ReasonUnknownError, "failed to resolve owner reference: %s", err.Error())

		logger.Info("failed to fetch signing key owner", "error", err.Error())

		return err
	}

	if err := r.validateOwnerRequirements(ownerObj, signingKey); err != nil {
		signingKey.Status.MarkOwnerResolveFailed(v1alpha1.ReasonNotAllowed, "SigningKey does not match selector requirements: %w", err)

		return TerminalError(err)
	}

	ownerAPIVersion, ownerKind := ownerObj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
	signingKey.Status.MarkOwnerResolved(v1alpha1.TypedObjectReference{
		APIVersion: ownerAPIVersion,
		Kind:       ownerKind,
		Name:       ownerObj.GetName(),
		Namespace:  ownerObj.GetNamespace(),
		UID:        ownerObj.GetUID(),
	})

	return nil
}

func (r *SigningKeyReconciler) validateOwnerRequirements(owner client.Object, signingKey *v1alpha1.SigningKey) error {
	if owner.GetNamespace() != signingKey.GetNamespace() {
		return fmt.Errorf("owner namespace %q does not match signing key namespace %q", owner.GetNamespace(), signingKey.GetNamespace())
	}

	var labelSelector *metav1.LabelSelector

	switch owner := owner.(type) {
	case *v1alpha1.Operator:
		labelSelector = owner.Spec.SigningKeysSelector
	case *v1alpha1.Account:
		labelSelector = owner.Spec.SigningKeysSelector
	default:
		return fmt.Errorf("unsupported owner kind evaluating owner requirements: %T", owner)
	}

	if labelSelector == nil {
		return nil
	}

	s, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return fmt.Errorf("failed to parse signing key label selector: %w", err)
	}

	if !s.Matches(labels.Set(signingKey.GetLabels())) {
		return fmt.Errorf("signing key does not match selector requirements")
	}

	return nil
}

func (r *SigningKeyReconciler) reconcileLabels(ctx context.Context, signingKey *v1alpha1.SigningKey) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	if signingKey.Status.OwnerRef == nil {
		return reconcile.Result{}, nil
	}

	if signingKey.Labels == nil {
		signingKey.Labels = make(map[string]string)
	}

	var needsUpdate bool

	switch signingKey.Status.OwnerRef.Kind {
	case "Operator":
		if signingKey.Labels[resources.LabelOperatorName] != signingKey.Status.OwnerRef.Name {
			signingKey.Labels[resources.LabelOperatorName] = signingKey.Status.OwnerRef.Name
			needsUpdate = true
		}
	case "Account":
		if signingKey.Labels[resources.LabelAccountName] != signingKey.Status.OwnerRef.Name {
			signingKey.Labels[resources.LabelAccountName] = signingKey.Status.OwnerRef.Name
			needsUpdate = true
		}
	default:
		logger.Error(
			fmt.Errorf("unsupported owner kind"),
			"expected Operator or Account kind for SigningKey owner",
			"kind", signingKey.Status.OwnerRef.Kind)
	}

	if !needsUpdate {
		return reconcile.Result{}, nil
	}

	if err := r.Client.Update(ctx, signingKey); err != nil {
		logger.Error(err, "failed to update signing key labels")

		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SigningKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SigningKey{}).
		Owns(&v1.Secret{}).
		Watches(&v1alpha1.Operator{}, signingKeyOperatorWatcher(mgr.GetLogger(), mgr.GetClient())).
		Watches(&v1alpha1.Account{}, signingKeyAccountWatcher(mgr.GetLogger(), mgr.GetClient())).
		Complete(r)
}


// signingKeyOperatorWatcher will enqueue any SigningKeys which are managed by the Operator being
// watched.
// Similar to accountOperatorWatcher, this will almost always be every SigningKey in the cluster
// unless the environment contains multiple Operators.
func signingKeyOperatorWatcher(logger logr.Logger, c client.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		operator, ok := obj.(*v1alpha1.Operator)
		if !ok {
			logger.Info("Operator watcher received non-Operator object",
				"kind", obj.GetObjectKind().GroupVersionKind().String())
			return nil
		}

		var signingKeyList v1alpha1.SigningKeyList

		if err := c.List(ctx, &signingKeyList, client.InNamespace(operator.Namespace), client.MatchingLabels{
			resources.LabelOperatorName: operator.Name,
		}); err != nil {
			logger.Error(err, "failed to list accounts for operator during enqueue handler", "operator", operator.Name)

			return nil
		}

		requests := make([]reconcile.Request, len(signingKeyList.Items))

		for i, acc := range signingKeyList.Items {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      acc.Name,
					Namespace: acc.Namespace,
				},
			}
		}

		return requests
	})
}

// signingKeyAccountWatcher will enqueue any SigningKeys which are managed by the Account being
// watched.
func signingKeyAccountWatcher(logger logr.Logger, c client.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		account, ok := obj.(*v1alpha1.Account)
		if !ok {
			logger.Info("Account watcher received non-Account object",
				"kind", obj.GetObjectKind().GroupVersionKind().String())
			return nil
		}

		var signingKeyList v1alpha1.SigningKeyList

		if err := c.List(ctx, &signingKeyList, client.InNamespace(account.Namespace), client.MatchingLabels{
			resources.LabelAccountName: account.Name,
		}); err != nil {
			logger.Error(err, "failed to list signing keys for account during enqueue handler", "account", account.Name)

			return nil
		}

		requests := make([]reconcile.Request, len(signingKeyList.Items))

		for i, acc := range signingKeyList.Items {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      acc.Name,
					Namespace: acc.Namespace,
				},
			}
		}

		return requests
	})
}
