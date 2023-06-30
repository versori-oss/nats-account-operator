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
	"github.com/versori-oss/nats-account-operator/pkg/apis"
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
	SigningKey SigningKeyReference `json:"signingKey"`

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

	// Identities is a JWT claim for the Account.
	Identities []Identity `json:"identities,omitempty"`

	// Limits is a JWT claim for the Account.
	Limits *AccountLimits `json:"limits"`
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

type AccountLimits struct {
	Subs      int64 `json:"subs"`
	Conn      int64 `json:"conn"`
	Leaf      int64 `json:"leaf"`
	Imports   int64 `json:"imports"`
	Exports   int64 `json:"exports"`
	Data      int64 `json:"data"`
	Payload   int64 `json:"payload"`
	Wildcards bool  `json:"wildcards"`
}

// AccountStatus defines the observed state of Account
type AccountStatus struct {
	KeyPair     *KeyPair                   `json:"keyPair,omitempty"`
	SigningKeys []SigningKeyEmbeddedStatus `json:"signingKeys,omitempty"`
	OperatorRef *InferredObjectReference   `json:"operatorRef,omitempty"`
	Conditions  apis.Conditions            `json:"conditions,omitempty"`
}

func (s *AccountStatus) GetConditions() apis.Conditions {
	return s.Conditions
}

func (s *AccountStatus) SetConditions(conditions apis.Conditions) {
	s.Conditions = conditions
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
