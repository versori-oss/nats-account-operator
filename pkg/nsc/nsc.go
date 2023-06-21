// Package nsc contains useful functions for pushing/updating account JWTs to the server.
// These functions also handle resolving the operator/system accounts Custom Resources on the cluster.
package nsc

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type NSCInterface interface {
	PushJWT(ctx context.Context, ajwt string) error
	UpdateJWT(ctx context.Context, accId string, ajwt string) error
}

var _ NSCInterface = (*NscHelper)(nil)

// NscHelper is a struct that contains the necessary information to push JWTs to the server
type NscHelper struct {
	OperatorRef  *v1alpha1.InferredObjectReference
	AccClientSet accountsclientsets.AccountsV1alpha1Interface
	CV1Interface corev1.CoreV1Interface
}

// PushJWT pushes a JWT to the server. It uses the accClientSet to resolve the operator account and then the system account.
// It then creates a NATS client using the retrieved system account credentials
func (n *NscHelper) PushJWT(ctx context.Context, ajwt string) error {
	operator, sysAcc, err := n.getOperatorAndSysAccount(ctx)
	if err != nil {
		return err
	}

	sysAccJWT, sysAccSeed, err := n.getSysAccountJWTSeed(ctx, sysAcc)
	if err != nil {
		return err
	}

	// TODO @JoeLanglands check this is the correct server URL
	serverUrl := operator.Spec.AccountServerURL

	accJWTSeedOpt := nats.UserJWTAndSeed(sysAccJWT, sysAccSeed)
	nc, err := nats.Connect(serverUrl, accJWTSeedOpt)
	if err != nil {
		return err
	}
	natsClient := NewNatsClient(nc)
	defer natsClient.Close()

	err = natsClient.PushAccountJWT(ctx, ajwt)
	if err != nil {
		return err
	}

	return nil
}

// UpdateJWT updates the JWT owned by the account identified by accId. Works similarly to PushJWT.
func (n *NscHelper) UpdateJWT(ctx context.Context, accId string, ajwt string) error {
	operator, sysAcc, err := n.getOperatorAndSysAccount(ctx)
	if err != nil {
		return err
	}

	sysAccJWT, sysAccSeed, err := n.getSysAccountJWTSeed(ctx, sysAcc)
	if err != nil {
		return err
	}

	serverUrl := operator.Spec.AccountServerURL

	accJWTSeedOpt := nats.UserJWTAndSeed(sysAccJWT, sysAccSeed)
	nc, err := nats.Connect(serverUrl, accJWTSeedOpt)
	if err != nil {
		return err
	}
	natsClient := NewNatsClient(nc)
	defer natsClient.Close()

	err = natsClient.UpdateAccountJWT(ctx, "", ajwt)
	if err != nil {
		return err
	}

	return nil
}

// getOperatorAndSysAccount retrieves the operator and system account from the cluster using the accClientSet.
func (n *NscHelper) getOperatorAndSysAccount(ctx context.Context) (*v1alpha1.Operator, *v1alpha1.Account, error) {
	operator, err := n.AccClientSet.Operators(n.OperatorRef.Namespace).Get(ctx, n.OperatorRef.Name, v1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	sysAcc, err := n.AccClientSet.Accounts(operator.Status.ResolvedSystemAccount.Namespace).Get(ctx, operator.Status.ResolvedSystemAccount.Name, v1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	return operator, sysAcc, nil
}

// TODO @JoeLanglands Is this all even possible? Because I'm not sure if the jwt's will exist on creation. Sure the operator/sys account will
// resolve ok but are the JWT's there? Mainly in the case where the account here IS the system account. Just gonna continue writing it and then
// will tackle that problem when I get to it. Maybe you have to bootstrap the cluster with system account/operator secrets idk.
// Also need to make sure the below function works properly

// getSysAccountJWTSeed retrieves the JWT and seed for the system account. The order of the returned values are: JWT, Seed, error.
func (n *NscHelper) getSysAccountJWTSeed(ctx context.Context, acc *v1alpha1.Account) (jwt string, seed string, err error) {
	jwtSecret, err := n.CV1Interface.Secrets(acc.GetNamespace()).Get(ctx, acc.Spec.JWTSecretName, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	seedSecret, err := n.CV1Interface.Secrets(acc.GetNamespace()).Get(ctx, acc.Spec.SeedSecretName, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	seedBytes, ok := seedSecret.Data[v1alpha1.NatsSecretSeedKey]
	if !ok {
		return "", "", errors.New("seed secret does not contain seed key")
	}

	jwtBytes, ok := jwtSecret.Data[v1alpha1.NatsSecretJWTKey]
	if !ok {
		return "", "", errors.New("jwt secret does not contain jwt key")
	}

	return string(jwtBytes), string(seedBytes), err
}

// TODO @JoeLanglands I know you attempted to make this nice and generic to work with accounts and operators but it might only need to deal with Accounts.
// getJWTClaims attempts to retrieve the JWT claims from the object's JWT secret. The object can either be an v1alpha1.Account or a v1alpha1.Operator.
// func (n *NatsPusher) getJWTClaims(ctx context.Context, obj client.Object) (*jwt.Claims, error) {
// 	var err error
// 	var claims jwt.Claims

// 	switch obj := obj.(type) {
// 	case *v1alpha1.Operator:
// 		jwtSecret, err := n.CV1Interface.Secrets(obj.GetNamespace()).Get(ctx, obj.Spec.JWTSecretName, v1.GetOptions{})
// 		if err != nil {
// 			return nil, err
// 		}
// 		objJWT := string(jwtSecret.Data["jwt"])
// 		claims, err = jwt.DecodeOperatorClaims(objJWT)
// 		return &claims, err
// 	case *v1alpha1.Account:
// 		jwtSecret, err := n.CV1Interface.Secrets(obj.GetNamespace()).Get(ctx, obj.Spec.JWTSecretName, v1.GetOptions{})
// 		if err != nil {
// 			return nil, err
// 		}
// 		objJWT := string(jwtSecret.Data["jwt"])
// 		claims, err = jwt.DecodeAccountClaims(objJWT)
// 		return &claims, err
// 	default:
// 		err = errors.New("unknown object type")
// 		return nil, err
// 	}
// }
