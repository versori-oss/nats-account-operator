package nsc

import (
	"fmt"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

func CreateUserClaims(
	resource *v1alpha1.User,
	account *v1alpha1.Account,
	signingKey nkeys.KeyPair,
) (claims *jwt.UserClaims, ujwt string, err error) {
	claims = jwt.NewUserClaims(resource.Status.KeyPair.PublicKey)
	claims.Name = resource.Name

	spec := resource.Spec
	specLimits := spec.Limits

	claims.Limits = jwt.Limits{
		UserLimits: jwt.UserLimits{
			Src:    specLimits.Src,
			Times:  ConvertToNatsTimeRanges(specLimits.Times),
			Locale: specLimits.Locale,
		},
		NatsLimits: ConvertToNatsLimits(specLimits.NatsLimits, claims.Limits.NatsLimits),
	}

	if spec.BearerToken != nil {
		claims.BearerToken = *spec.BearerToken
	}

	if spec.Permissions != nil {
		claims.UserPermissionLimits = jwt.UserPermissionLimits{
			Permissions: jwt.Permissions{
				Pub: jwt.Permission{
					Allow: spec.Permissions.Pub.Allow,
					Deny:  spec.Permissions.Pub.Deny,
				},
				Sub: jwt.Permission{
					Allow: spec.Permissions.Sub.Allow,
					Deny:  spec.Permissions.Sub.Deny,
				},
			},
			Limits:                 jwt.Limits{},
			BearerToken:            false,
			AllowedConnectionTypes: nil,
		}

		if spec.Permissions.Resp != nil {
			claims.Resp = &jwt.ResponsePermission{
				MaxMsgs: spec.Permissions.Resp.MaxMsgs,
				Expires: spec.Permissions.Resp.TTL.Duration,
			}
		}
	}

	skPub, err := signingKey.PublicKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get public key from key pair: %w", err)
	}

	// if the issuer is a signing key and not a direct account, we must set the issuer_account
	// field to the public key of the account that owns the signing key.
	accountKP := account.Status.KeyPair
	if accountKP != nil && accountKP.PublicKey != skPub {
		claims.IssuerAccount = accountKP.PublicKey
	}

	ujwt, err = claims.Encode(signingKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode account claims: %w", err)
	}

	return claims, ujwt, nil
}
