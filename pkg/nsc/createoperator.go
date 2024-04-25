package nsc

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

func CreateOperatorClaims(resource *v1alpha1.Operator, signingKey nkeys.KeyPair) (*jwt.OperatorClaims, string, error) {
	spec := resource.Spec

	if resource.Status.ResolvedSystemAccount == nil {
		return nil, "", fmt.Errorf("cannot create operator without a resolved system account")
	}

	if resource.Status.KeyPair == nil {
		return nil, "", fmt.Errorf("cannot create operator without a key pair")
	}

	signingKeys := make([]string, len(resource.Status.SigningKeys))
	for i, sk := range resource.Status.SigningKeys {
		signingKeys[i] = sk.KeyPair.PublicKey
	}

	claims := jwt.NewOperatorClaims(resource.Status.KeyPair.PublicKey)

	claims.Name = resource.Name
	claims.IssuedAt = time.Now().Unix()
	claims.Operator = jwt.Operator{
		SigningKeys:         signingKeys,
		AccountServerURL:    spec.AccountServerURL,
		OperatorServiceURLs: spec.OperatorServiceURLs,
		SystemAccount:       resource.Status.ResolvedSystemAccount.PublicKey,
		GenericFields: jwt.GenericFields{
			Type: jwt.OperatorClaim,
		},
	}

	ojwt, err := claims.Encode(signingKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode operator claims: %w", err)
	}

	return claims, ojwt, nil
}
