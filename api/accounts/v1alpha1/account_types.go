/*
MIT License

Copyright (c) 2022 Versori Ltd

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ImportExportType string

const (
	ImportExportTypeStream  ImportExportType = "stream"
	ImportExportTypeService ImportExportType = "service"
)

type ResponseType string

const (
	ResponseTypeSingleton ResponseType = "singleton"
	ResponseTypeStream    ResponseType = "stream"
	ResponseTypeChunked   ResponseType = "chunked"
)

// AccountSpec defines the desired state of Account
type AccountSpec struct {
	// SigningKey is the reference to the SigningKey that will be used to sign JWTs for this Account. The controller
	// will check the owner of the SigningKey is an Operator, and that this Account can be managed by that Operator
	// following its namespace and label selector restrictions.
	Issuer IssuerReference `json:"issuer"`

	// UsersNamespaceSelector defines which namespaces are allowed to contain Users managed by this Account. The default
	// restricts to the same namespace as the Account, it can be set to an empty selector `{}` to allow all namespaces.
	UsersNamespaceSelector *metav1.LabelSelector `json:"usersNamespaceSelector,omitempty"`

	// UsersSelector defines which Users are allowed to be managed by this Account. The default implies no label
	// selector and all User resources will be allowed (subject to the UsersNamespaceSelector above).
	UsersSelector *metav1.LabelSelector `json:"usersSelector,omitempty"`

	// JWTSecretName is the name of the Secret that will be created to hold the JWT signing key for this Account.
	JWTSecretName string `json:"jwtSecretName"`

	// SeedSecretName is the name of the Secret that will be created to hold the seed for this Account.
	SeedSecretName string `json:"seedSecretName"`

	// SigningKeysSelector is the label selector to restrict which SigningKeys can be used to sign JWTs for this
	// Account. SigningKeys must be in the same namespace as the Account.
	SigningKeysSelector *metav1.LabelSelector `json:"signingKeysSelector,omitempty"`

	// Imports is a JWT claim for the Account.
	Imports []AccountImport `json:"imports,omitempty"`

	// Exports is a JWT claim for the Account.
	Exports []AccountExport `json:"exports,omitempty"`

	// Limits is a JWT claim for the Account.
	Limits *OperatorLimits `json:"limits,omitempty"`
}

type AccountImport struct {
	Name    string           `json:"name"`
	Subject string           `json:"subject"`
	Account string           `json:"account"`
	Token   string           `json:"token"`
	To      string           `json:"to"`
	Type    ImportExportType `json:"type"`
}

type AccountExport struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	// Type is the type of export. This must be one of "stream" or "service".
	Type     ImportExportType `json:"type"`
	TokenReq bool             `json:"tokenReq"`
	// ResponseType is the type of response that will be sent to the requestor. This must be one of
	// "singleton", "stream" or "chunked" if Type is "service". If Type is "stream", this must be left as an empty string.
	ResponseType         ResponseType           `json:"responseType"`
	ServiceLatency       *AccountServiceLatency `json:"serviceLatency,omitempty"`
	AccountTokenPosition uint                   `json:"accountTokenPosition"`
}

type AccountServiceLatency struct {
	Sampling int    `json:"sampling"`
	Results  string `json:"results"`
}

// OperatorLimits are used to limit access by an account
type OperatorLimits struct {
	Nats      NatsLimits      `json:"nats,omitempty"`
	Account   AccountLimits   `json:"account,omitempty"`
	JetStream JetStreamLimits `json:"jetStream,omitempty"`
}

type NatsLimits struct {
	Subs    *int64 `json:"subs,omitempty"`    // Max number of subscriptions
	Data    *int64 `json:"data,omitempty"`    // Max number of bytes
	Payload *int64 `json:"payload,omitempty"` // Max message payload
}

type AccountLimits struct {
	Imports         *int64 `json:"imports,omitempty"`        // Max number of imports
	Exports         *int64 `json:"exports,omitempty"`        // Max number of exports
	WildcardExports *bool  `json:"wildcards,omitempty"`      // Are wildcards allowed in exports
	DisallowBearer  bool   `json:"disallowBearer,omitempty"` // User JWT can't be bearer token
	Conn            *int64 `json:"conn,omitempty"`           // Max number of active connections
	LeafNodeConn    *int64 `json:"leaf,omitempty"`           // Max number of active leaf node connections
}

type JetStreamLimits struct {
	MemoryStorage        int64 `json:"memoryStorage,omitempty"`        // Max number of bytes stored in memory across all streams. (0 means disabled)
	DiskStorage          int64 `json:"diskStorage,omitempty"`          // Max number of bytes stored on disk across all streams. (0 means disabled)
	Streams              int64 `json:"streams,omitempty"`              // Max number of streams
	Consumer             int64 `json:"consumer,omitempty"`             // Max number of consumers
	MaxAckPending        int64 `json:"maxAckPending,omitempty"`        // Max ack pending of a Stream
	MemoryMaxStreamBytes int64 `json:"memoryMaxStreamBytes,omitempty"` // Max bytes a memory backed stream can have. (0 means disabled/unlimited)
	DiskMaxStreamBytes   int64 `json:"diskMaxStreamBytes,omitempty"`   // Max bytes a disk backed stream can have. (0 means disabled/unlimited)
	MaxBytesRequired     bool  `json:"maxBytesRequired,omitempty"`     // Max bytes required by all Streams
}

// AccountStatus defines the observed state of Account
type AccountStatus struct {
	Status `json:",inline"`

	KeyPair     *KeyPair                   `json:"keyPair,omitempty"`
	SigningKeys []SigningKeyEmbeddedStatus `json:"signingKeys,omitempty"`
	OperatorRef *InferredObjectReference   `json:"operatorRef,omitempty"`
}

type OperatorRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Public Key",type=string,JSONPath=`.status.keyPair.publicKey`
//+kubebuilder:printcolumn:name="Operator",type=string,JSONPath=`.status.operatorRef.name`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`

// Account is the Schema for the accounts API
type Account struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AccountSpec   `json:"spec,omitempty"`
	Status AccountStatus `json:"status,omitempty"`
}

var _ KeyPairable = (*Account)(nil)

func (a *Account) GetStatus() *Status {
	return &a.Status.Status
}

func (a *Account) GetKeyPair() *KeyPair {
	return a.Status.KeyPair
}

//+kubebuilder:object:root=true

// AccountList contains a list of Account
type AccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Account `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Account{}, &AccountList{})
}
