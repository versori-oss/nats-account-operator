package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

type ReplyData struct {
	Account     string `json:"account"`
	Code        int    `json:"code"`
	Description string `json:"description"`
}

type UpdateReply struct {
	Data       *ReplyData     `json:"data,omitempty"`
	Error      *ReplyData     `json:"error,omitempty"`
	ServerInfo map[string]any `json:"server"`
}

type ErrAccountJWTNotPushed struct {
	msg string
}

func (e *ErrAccountJWTNotPushed) Error() string { return e.msg }

type NatsClient struct {
	conn *nats.Conn
}

func NewNatsClient(conn *nats.Conn) *NatsClient {
	return &NatsClient{
		conn: conn,
	}
}

func (n *NatsClient) PushAccountJWT(ctx context.Context, ajwt string) error {
	msg, err := n.conn.RequestWithContext(ctx, "$SYS.REQ.CLAIMS.UPDATE", []byte(ajwt))
	if err != nil {
		return err
	}
	return checkReplyForError(msg.Data)
}

func (n *NatsClient) GetAccountJWT(ctx context.Context, accountID string) (string, error) {
	subject := fmt.Sprintf("$SYS.REQ.ACCOUNT.%s.CLAIMS.LOOKUP", accountID)
	msg, err := n.conn.RequestWithContext(ctx, subject, nil)
	if err != nil {
		return "", err
	}
	if len(msg.Data) == 0 {
		return "", &ErrAccountJWTNotPushed{msg: "account jwt not pushed"}
	}
	return string(msg.Data), nil
}

func checkReplyForError(msg []byte) error {
	var msgData UpdateReply
	err := json.Unmarshal(msg, &msgData)
	if err != nil {
		return err
	}
	if msgData.Error != nil {
		return errors.New(msgData.Error.Description)
	}
	return nil
}
