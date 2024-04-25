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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/versori-oss/nats-account-operator/pkg/apis"
)

// TLSConfig is the TLS configuration for communicating to the NATS server for pushing/deleting account JWTs.
// Initially this only supports defining server-side TLS verification by defining a CA certificate within a secret, in
// the future we will support mutual-TLS authentication by defining a client certificate and key within a secret.
type TLSConfig struct {
	// CAFile is a reference to a secret containing the CA certificate to use for TLS connections.
	CAFile *v1.SecretKeySelector `json:"caFile,omitempty"`
}

// OperatorSpec defines the desired state of Operator
type OperatorSpec struct {
	// JWTSecretName is the name of the secret containing the self-signed Operator JWT.
	JWTSecretName string `json:"jwtSecretName"`

	// SeedSecretName is the name of the secret containing the seed for this Operator.
	SeedSecretName string `json:"seedSecretName"`

	// AccountsNamespaceSelector defines which namespaces are allowed to contain Accounts managed by this Operator. By
	// default, the Operator will manage Accounts in the same namespace as the Operator, it can be set to an empty
	// selector `{}` to allow all namespaces.
	AccountsNamespaceSelector *metav1.LabelSelector `json:"accountsNamespaceSelector,omitempty"`

	// AccountsSelector allows the Operator to restrict the Accounts it manages to those matching the selector. The
	// default (`null`) and `{}` selectors are equivalent and match all Accounts. This is used in combination to the
	// AccountsNamespaceSelector.
	AccountsSelector *metav1.LabelSelector `json:"accountsSelector,omitempty"`

	// SigningKeysSelector allows the Operator to restrict the SigningKeys it manages to those matching the selector.
	// Only SigningKeys in the same namespace as the Operator are considered. The default (`null`) and `{}` selectors
	// are equivalent and match all SigningKeys.
	SigningKeysSelector *metav1.LabelSelector `json:"signingKeysSelector,omitempty"`

	// SystemAccountRef is a reference to the Account that this Operator will use as it's system account. It must exist
	// in the same namespace as the Operator, the AccountsNamespaceSelector and AccountsSelector are ignored.
	SystemAccountRef v1.LocalObjectReference `json:"systemAccountRef"`

	// TLSConfig is the TLS configuration for communicating to the NATS server for pushing/deleting account JWTs.
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`

	// AccountServerURL is a JWT claim for the Operator
	AccountServerURL string `json:"accountServerURL,omitempty"`

	// OperatorServiceURLs is a JWT claim for the Operator
	OperatorServiceURLs []string `json:"operatorServiceURLs,omitempty"`
}

// OperatorStatus defines the observed state of Operator
type OperatorStatus struct {
	Status `json:",inline"`

	// KeyPair is the public/private key pair for the Operator. This is created by the controller when an Operator is
	// created.
	KeyPair *KeyPair `json:"keyPair,omitempty"`

	// SigningKeys is the list of additional SigningKey resources which are owned by this Operator. Accounts may be
	// created using the default KeyPair or any of these SigningKeys.
	SigningKeys []SigningKeyEmbeddedStatus `json:"signingKeys,omitempty"`

	// ResolvedSystemAccount is the Account that this Operator will use as it's system account. This is the same as the
	// resource defined in OperatorSpec.SystemAccountRef, but validated that the resource exists.
	ResolvedSystemAccount *KeyPairReference `json:"resolvedSystemAccount,omitempty"`
}

func (os *OperatorStatus) GetConditions() apis.Conditions {
	return os.Conditions
}

func (os *OperatorStatus) SetConditions(conditions apis.Conditions) {
	os.Conditions = conditions
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=nop;natsoperator
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Public Key",type=string,JSONPath=`.status.keyPair.publicKey`
//+kubebuilder:printcolumn:name="System Account",type=string,JSONPath=`.status.resolvedSystemAccount.name`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`

// Operator is the Schema for the operators API
type Operator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorSpec   `json:"spec,omitempty"`
	Status OperatorStatus `json:"status,omitempty"`
}

var _ KeyPairable = (*Operator)(nil)

func (o *Operator) GetStatus() *Status {
	return &o.Status.Status
}

func (o *Operator) GetKeyPair() *KeyPair {
	return o.Status.KeyPair
}

//+kubebuilder:object:root=true

// OperatorList contains a list of Operator
type OperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Operator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Operator{}, &OperatorList{})
}
