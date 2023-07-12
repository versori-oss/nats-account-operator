package nsc

import (
	"context"
	"fmt"
	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
	clientsetv1alpha1 "github.com/versori-oss/nats-account-operator/pkg/generated/clientset/versioned/typed/accounts/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientsetv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type SystemAccountLoader struct {
	accounts clientsetv1alpha1.AccountsV1alpha1Interface
	core     clientsetv1.CoreV1Interface
}

func NewSystemAccountLoader(
	accounts clientsetv1alpha1.AccountsV1alpha1Interface,
	core clientsetv1.CoreV1Interface,
) *SystemAccountLoader {
	return &SystemAccountLoader{
		accounts: accounts,
		core:     core,
	}
}

func (s *SystemAccountLoader) Load(ctx context.Context, operator *v1alpha1.Operator) (seed []byte, err error) {
	// TODO: figure out how to handle errors here

	if operator.Status.ResolvedSystemAccount == nil {
		return nil, fmt.Errorf("operator %s/%s does not have a resolved system account", operator.Namespace, operator.Name)
	}

	account, err := s.accounts.Accounts(operator.Namespace).Get(ctx, operator.Status.ResolvedSystemAccount.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if account.Status.KeyPair == nil {
		return nil, fmt.Errorf("system account %s/%s does not have a keypair", account.Namespace, account.Name)
	}

	seedSecret, err := s.core.Secrets(account.Namespace).Get(ctx, account.Status.KeyPair.SeedSecretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	seedBytes, ok := seedSecret.Data[v1alpha1.NatsSecretSeedKey]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s is invalid, missing field: %s", seedSecret.Namespace, seedSecret.Name, v1alpha1.NatsSecretSeedKey)
	}

	return seedBytes, nil
}
