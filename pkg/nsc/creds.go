package nsc

import (
	"time"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nkeys"
)

type ClaimOption func(cd *jwt.ClaimsData) error

func ExpiresAt(t time.Time) ClaimOption {
	return func(cd *jwt.ClaimsData) error {
		cd.Expires = t.Unix()
		return nil
	}
}

func CreateUser(name string, payload jwt.User, signingKey nkeys.KeyPair, opts ...ClaimOption) (ujwt string, pubKey string, seed []byte, err error) {
	kp, err := nkeys.CreateUser()
	if err != nil {
		return "", "", nil, err
	}

	seed, err = kp.Seed()
	if err != nil {
		return "", "", nil, err
	}

	pubKey, err = kp.PublicKey()
	if err != nil {
		return "", "", nil, err
	}

	claims := jwt.NewUserClaims(pubKey)

	claims.User = payload
	claims.Name = name

	claims.IssuerAccount, err = signingKey.PublicKey()
	if err != nil {
		return "", "", nil, err
	}

	for _, fn := range opts {
		if err = fn(&claims.ClaimsData); err != nil {
			return "", "", nil, err
		}
	}

	ujwt, err = claims.Encode(signingKey)
	if err != nil {
		return "", "", nil, err
	}

	return ujwt, pubKey, seed, nil
}

func CreateAccount(name string, payload jwt.Account, signingKey nkeys.KeyPair, opts ...ClaimOption) (ajwt string, pubKey string, seed []byte, err error) {
	kp, err := nkeys.CreateAccount()
	if err != nil {
		return "", "", nil, err
	}

	seed, err = kp.Seed()
	if err != nil {
		return "", "", nil, err
	}

	pubKey, err = kp.PublicKey()
	if err != nil {
		return "", "", nil, err
	}

	claims := jwt.NewAccountClaims(pubKey)
	claims.Name = name
	claims.Account = payload

	for _, fn := range opts {
		if err = fn(&claims.ClaimsData); err != nil {
			return "", "", nil, err
		}
	}

	ajwt, err = claims.Encode(signingKey)
	if err != nil {
		return "", "", nil, err
	}

	return ajwt, pubKey, seed, nil
}

func UpdateAccount(existing jwt.AccountClaims, signingKey nkeys.KeyPair, opts ...ClaimOption) (ajwt string, err error) {
	for _, fn := range opts {
		if err = fn(&existing.ClaimsData); err != nil {
			return "", err
		}
	}

	ajwt, err = existing.Encode(signingKey)
	if err != nil {
		return "", err
	}

	return ajwt, nil
}
