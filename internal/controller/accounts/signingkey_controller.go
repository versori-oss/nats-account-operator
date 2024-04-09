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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
)

// SigningKeyReconciler reconciles a SigningKey object
type SigningKeyReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	CV1Interface      corev1.CoreV1Interface
	AccountsClientSet accountsclientsets.AccountsV1alpha1Interface
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
		if errors.IsNotFound(err) {
			logger.V(1).Info("signing key not found or not ready")
			return ctrl.Result{
				RequeueAfter: 30 * time.Second,
			}, nil
		}
		logger.Error(err, "failed to fetch signing key")
		return ctrl.Result{}, err
	}

	originalStatus := signingKey.Status.DeepCopy()

	signingKey.Status.InitializeConditions()

	defer func() {
		if !equality.Semantic.DeepEqual(*originalStatus, signingKey.Status) {
			if err2 := r.Status().Update(ctx, signingKey); err2 != nil {
				logger.Error(err2, "failed to update signing key status")

				err = multierr.Append(err, err2)
			}
		}
	}()

	if err := r.ensureOwnerResolved(ctx, signingKey); err != nil {
		logger.V(1).Info("failed to resolve owner for signing key", "name", signingKey.Name)
		signingKey.Status.MarkOwnerResolveFailed("failed to resolve owner", "%s:%s", signingKey.Spec.OwnerRef.Name, signingKey.Spec.OwnerRef.Kind)
		return ctrl.Result{}, err
	}

	if err := r.ensureKeyPair(ctx, signingKey); err != nil {
		logger.Error(err, "failed to ensure key pair")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *SigningKeyReconciler) ensureKeyPair(ctx context.Context, signingKey *v1alpha1.SigningKey) error {
	logger := log.FromContext(ctx)

	var publicKey string
	secret, err := r.CV1Interface.Secrets(signingKey.Namespace).Get(ctx, signingKey.Spec.SeedSecretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		var keyPair nkeys.KeyPair
		switch signingKey.Spec.OwnerRef.Kind {
		case v1alpha1.SigningKeyTypeAccount:
			keyPair, err = nkeys.CreateAccount()
		case v1alpha1.SigningKeyTypeOperator:
			keyPair, err = nkeys.CreateOperator()
		default:
			err := errors.NewBadRequest(fmt.Sprintf("unknown owner kind: %s", signingKey.Spec.OwnerRef.Kind))
			return err
		}
		if err != nil {
			logger.Error(err, "failed to create key pair")
			return err
		}

		seed, err := keyPair.Seed()
		if err != nil {
			logger.Error(err, "failed to get seed")
			return err
		}
		publicKey, err = keyPair.PublicKey()
		if err != nil {
			logger.Error(err, "failed to get public key")
			return err
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
			return err
		}

		_, err = r.CV1Interface.Secrets(signingKey.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
		if err != nil {
			logger.Error(err, "failed to create seed secret")
			return err
		}
	} else if err != nil {
		logger.Error(err, "failed to fetch seed secret")
		return err
	} else {
		publicKey = string(secret.Data[v1alpha1.NatsSecretPublicKeyKey])
	}

	signingKey.Status.MarkSeedSecretReady(publicKey, signingKey.Spec.SeedSecretName)

	return nil
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

// SetupWithManager sets up the controller with the Manager.
func (r *SigningKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SigningKey{}).
		Owns(&v1.Secret{}).
		Complete(r)
}
