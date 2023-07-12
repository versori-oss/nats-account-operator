package helpers

import "github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"

func IsSystemAccount(account *v1alpha1.Account, operator *v1alpha1.Operator) bool {
	return account.Namespace == operator.Namespace && account.Name == operator.Spec.SystemAccountRef.Name
}
