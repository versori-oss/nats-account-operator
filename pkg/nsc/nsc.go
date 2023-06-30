// Package nsc contains useful functions for pushing/updating account JWTs to the server.
// These functions also handle resolving the operator/system accounts Custom Resources on the cluster.
package nsc

import (
	"context"
	"fmt"

	natsjwt "github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	accountsclientsets "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NSCInterface interface {
	PushJWT(ctx context.Context, ajwt string) error
	GetJWT(ctx context.Context, accId string) (string, error)
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
	logger := log.FromContext(ctx)

	operator, sysAcc, err := n.getOperatorAndSysAccount(ctx)
	if err != nil {
		return err
	}

	sysUsrJWT, sysUsrSeed, err := n.getSysUsrJWTSeed(ctx, sysAcc)
	if err != nil {
		logger.Error(err, "failed to get system user JWT and seed")
		return err
	}

	serverUrl := operator.Spec.AccountServerURL

	nc, err := nats.Connect(serverUrl, nats.UserJWTAndSeed(sysUsrJWT, sysUsrSeed))
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
	logger := log.FromContext(ctx)

	operator, sysAcc, err := n.getOperatorAndSysAccount(ctx)
	if err != nil {
		return err
	}

	sysUsrJWT, sysUsrSeed, err := n.getSysUsrJWTSeed(ctx, sysAcc)
	if err != nil {
		logger.Error(err, "failed to get system user JWT and seed")
		return err
	}

	serverUrl := operator.Spec.AccountServerURL

	nc, err := nats.Connect(serverUrl, nats.UserJWTAndSeed(sysUsrJWT, sysUsrSeed))
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

func (n *NscHelper) GetJWT(ctx context.Context, accId string) (string, error) {
	logger := log.FromContext(ctx)

	var ajwt string
	operator, sysAcc, err := n.getOperatorAndSysAccount(ctx)
	if err != nil {
		return "", err
	}

	sysUsrJWT, sysUsrSeed, err := n.getSysUsrJWTSeed(ctx, sysAcc)
	if err != nil {
		logger.Error(err, "failed to get system user JWT and seed")
		return "", err
	}

	serverUrl := operator.Spec.AccountServerURL

	nc, err := nats.Connect(serverUrl, nats.UserJWTAndSeed(sysUsrJWT, sysUsrSeed))
	if err != nil {
		return "", err
	}
	natsClient := NewNatsClient(nc)
	defer natsClient.Close()

	ajwt, err = natsClient.GetAccountJWT(ctx, accId)
	if err != nil {
		return "", err
	}
	return ajwt, nil
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

// getSysUsrJWTSeed retrieves the JWT and seed for a/the system account's user. The order of the returned values are: JWT, Seed, error.
func (n *NscHelper) getSysUsrJWTSeed(ctx context.Context, sysAcc *v1alpha1.Account) (jwt string, seed string, err error) {
	usrList, err := n.AccClientSet.Users(sysAcc.GetNamespace()).List(ctx, v1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	var sysUsr *v1alpha1.User
	for _, usr := range usrList.Items {
		if usr.Status.IsReady() && usr.Status.AccountRef.Name == sysAcc.GetName() {
			sysUsr = &usr
			break
		}
	}

	if sysUsr == nil {
		return "", "", fmt.Errorf("no system user available/ready for system account %s", sysAcc.GetName())
	}

	usrCredSecret, err := n.CV1Interface.Secrets(sysUsr.GetNamespace()).Get(ctx, sysUsr.Spec.CredentialsSecretName, v1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	usrCredentials := usrCredSecret.Data[v1alpha1.NatsSecretCredsKey]

	ujwt, err := natsjwt.ParseDecoratedJWT(usrCredentials)
	if err != nil {
		return "", "", err
	}

	seedKPair, err := natsjwt.ParseDecoratedUserNKey(usrCredentials)
	if err != nil {
		return "", "", err
	}

	seedBytes, err := seedKPair.Seed()
	if err != nil {
		return "", "", err
	}

	return ujwt, string(seedBytes), err
}
