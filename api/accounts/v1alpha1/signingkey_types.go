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

type SigningKeyType string

const (
	SigningKeyTypeOperator = "Operator"
	SigningKeyTypeAccount  = "Account"
)

// SigningKeySpec defines the desired state of SigningKey
type SigningKeySpec struct {
	// Type defines which prefix to use for the signing key, supported values are "Operator" and "Account".
	// +required
	Type SigningKeyType `json:"type"`

	// SeedSecretName is the name of the secret containing the seed for this signing key.
	// +required
	SeedSecretName string `json:"seedSecretName"`

	// OwnerRef references the owning object for this signing key. This should be one of Operator or Account. The
	// controller will validate that this SigningKey is allowed to be owned by the referenced resource by evaluating its
	// label selectors.
	OwnerRef SigningKeyOwnerReference `json:"ownerRef"`
}

// SigningKeyStatus defines the observed state of SigningKey
type SigningKeyStatus struct {
	// KeyPair contains the public and private key information for this signing key.
	KeyPair *KeyPair `json:"keyPair,omitempty"`

	// OwnerRef references the owning object for this signing key. This should be one of Operator or Account.
	OwnerRef *TypedObjectReference `json:"ownerRef,omitempty"`

	// Conditions contains the current status of the signing key.
	Conditions apis.Conditions `json:"conditions,omitempty"`
}

func (s *SigningKeyStatus) GetConditions() apis.Conditions {
	return s.Conditions
}

func (s *SigningKeyStatus) SetConditions(conditions apis.Conditions) {
	s.Conditions = conditions
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Public Key",type=string,JSONPath=`.status.keyPair.publicKey`
//+kubebuilder:printcolumn:name="Owner Kind",type=string,JSONPath=`.status.ownerRef.kind`
//+kubebuilder:printcolumn:name="Owner",type=string,JSONPath=`.status.ownerRef.name`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`

// SigningKey is the Schema for the signingkeys API
type SigningKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SigningKeySpec   `json:"spec,omitempty"`
	Status SigningKeyStatus `json:"status,omitempty"`
}

func (s *SigningKey) GetKeyPair() *KeyPair {
	return s.Status.KeyPair
}

//+kubebuilder:object:root=true

// SigningKeyList contains a list of SigningKey
type SigningKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SigningKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SigningKey{}, &SigningKeyList{})
}
