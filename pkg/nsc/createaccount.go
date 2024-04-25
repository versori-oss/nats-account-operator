package nsc

import (
	"fmt"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	"github.com/versori-oss/nats-account-operator/api/accounts/v1alpha1"
)

func CreateAccountClaims(
	resource *v1alpha1.Account,
	signingKey nkeys.KeyPair,
) (claims *jwt.AccountClaims, ajwt string, err error) {
	claims = jwt.NewAccountClaims(resource.Status.KeyPair.PublicKey)
	claims.Name = resource.Name

	spec := resource.Spec

	claims.Exports = ConvertToNATSExports(spec.Exports)
	claims.Imports = ConvertToNATSImports(spec.Imports)

	if spec.Limits != nil {
		claims.Limits = jwt.OperatorLimits{
			NatsLimits:    ConvertToNatsLimits(spec.Limits.Nats, claims.Limits.NatsLimits),
			AccountLimits: ConvertToAccountLimits(spec.Limits.Account, claims.Limits.AccountLimits),
			JetStreamLimits: jwt.JetStreamLimits{
				MemoryStorage:        spec.Limits.JetStream.MemoryStorage,
				DiskStorage:          spec.Limits.JetStream.DiskStorage,
				Streams:              spec.Limits.JetStream.Streams,
				Consumer:             spec.Limits.JetStream.Consumer,
				MaxAckPending:        spec.Limits.JetStream.MaxAckPending,
				MemoryMaxStreamBytes: spec.Limits.JetStream.MemoryMaxStreamBytes,
				DiskMaxStreamBytes:   spec.Limits.JetStream.DiskMaxStreamBytes,
				MaxBytesRequired:     spec.Limits.JetStream.MaxBytesRequired,
			},
		}
	}

	for _, sk := range resource.Status.SigningKeys {
		claims.SigningKeys.Add(sk.KeyPair.PublicKey)
	}

	ajwt, err = claims.Encode(signingKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode account claims: %w", err)
	}

	return claims, ajwt, nil
}
