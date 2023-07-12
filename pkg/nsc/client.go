package nsc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/versori-oss/nats-account-operator/pkg/nsc/internal"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	RequestSubjectClaimsUpdate = "$SYS.REQ.CLAIMS.UPDATE"
	RequestSubjectClaimsDelete = "$SYS.REQ.CLAIMS.DELETE"
)

type Client struct {
	conn *nats.Conn

	operator        nkeys.KeyPair
	operatorSubject string
}

func Connect(url string, operator nkeys.KeyPair, systemAccountSeed []byte, opts ...nats.Option) (*Client, error) {
	ujwt, useed, err := makeTemporaryUser(systemAccountSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary system account user: %w", err)
	}

	operatorPubkey, err := operator.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get operator public key: %w", err)
	}

	options := append(make([]nats.Option, 0, len(opts)+2), opts...)
	options = append(options,
		nats.UserJWTAndSeed(ujwt, useed),
		nats.Name("nats-account-operator"),
	)

	conn, err := nats.Connect(url, options...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:            conn,
		operator:        operator,
		operatorSubject: operatorPubkey,
	}, nil
}

func (c *Client) Push(ctx context.Context, jwt string) error {
	resp, err := c.do(ctx, RequestSubjectClaimsUpdate, []byte(jwt))
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("nats push failed: %s", resp.Error.Description)
	}

	log.FromContext(ctx).V(1).Info("nats push succeeded", "response_message", resp.Data.Message)

	return nil
}

func (c *Client) Delete(ctx context.Context, subject string) error {
	claims := jwt.NewGenericClaims(c.operatorSubject)
	claims.Data["accounts"] = []string{subject}

	payload, err := claims.Encode(c.operator)
	if err != nil {
		return fmt.Errorf("failed to encode jwt with operator key pair: %w", err)
	}

	resp, err := c.do(ctx, RequestSubjectClaimsDelete, []byte(payload))
	if err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("nats delete failed: %s", resp.Error.Description)
	}

	log.FromContext(ctx).V(1).Info("nats delete succeeded", "response_message", resp.Data.Message)

	return nil
}

func (c *Client) do(ctx context.Context, subj string, data []byte) (*internal.UpdateResponse, error) {
	resp, err := c.conn.RequestWithContext(ctx, subj, data)
	if err != nil {
		return nil, err
	}

	var reply internal.UpdateResponse
	if err := json.Unmarshal(resp.Data, &reply); err != nil {
		return nil, fmt.Errorf("failed to json unmarshal response: %w", err)
	}

	return &reply, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func makeTemporaryUser(accountSeed []byte) (ujwt string, seed string, err error) {
	sysKP, err := nkeys.FromSeed(accountSeed)
	if err != nil {
		return "", "", err
	}

	userKP, err := nkeys.CreateUser()
	if err != nil {
		return "", "", err
	}

	userPubkey, err := userKP.PublicKey()
	if err != nil {
		return "", "", err
	}

	userClaims := jwt.NewUserClaims(userPubkey)
	userClaims.Name = "k8s-operator-tmp-user"

	ujwt, err = userClaims.Encode(sysKP)
	if err != nil {
		return "", "", err
	}

	userSeed, err := userKP.Seed()
	if err != nil {
		return "", "", err
	}

	return ujwt, string(userSeed), nil
}
