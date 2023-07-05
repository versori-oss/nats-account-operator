package internal

import "time"

// ServerInfo is a copy of nats-server's ServerInfo struct, see:
//
// https://github.com/nats-io/nats-server/blob/eb2aa352ec357ee10e8ae71b046652ce680a0ee5/server/events.go#L184
type ServerInfo struct {
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	ID        string    `json:"id"`
	Cluster   string    `json:"cluster,omitempty"`
	Domain    string    `json:"domain,omitempty"`
	Version   string    `json:"ver"`
	Tags      []string  `json:"tags,omitempty"`
	Seq       uint64    `json:"seq"`
	JetStream bool      `json:"jetstream"`
	Time      time.Time `json:"time"`
}

type ErrorInfo struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Account     string `json:"account"`
}

type UpdateResponseData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Account string `json:"account"`
}

// UpdateResponse is the response payload from the $SYS.REQ.CLAIMS.UPDATE request. Error and Data are mutually
// exclusive.
type UpdateResponse struct {
	Server ServerInfo         `json:"server"`
	Error  *ErrorInfo         `json:"error,omitempty"`
	Data   UpdateResponseData `json:"data,omitempty"`
}
