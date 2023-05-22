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

// UserSpec defines the desired state of User
type UserSpec struct {
	// SigningKey is the reference to the SigningKey that will be used to sign JWTs for this User. The controller
	// will check the owner of the SigningKey is an Account, and that this User can be managed by that Account
	// following its namespace and label selector restrictions.
	SigningKey SigningKeyReference `json:"signingKey"`

	// JWTSecretName is the name of the Secret that will be created to store the JWT for this User.
	JWTSecretName string `json:"jwtSecretName"`

	// SeedSecretName is the name of the Secret that will be created to store the seed for this User.
	SeedSecretName string `json:"seedSecretName"`

	// CredentialsSecretName is the name of the Secret that will be created to store the credentials for this User.
	CredentialsSecretName string `json:"credentialsSecretName"`

	// Permissions is a JWT claim for the User.
	Permissions UserPermissions `json:"permissions"`

	// Limits is a JWT claim for the User.
	Limits UserLimits `json:"limits"`

	// BearerToken is a JWT claim for the User.
	BearerToken bool `json:"bearerToken"`
}

type UserPermissions struct {
	Pub  Permission     `json:"pub"`
	Sub  Permission     `json:"sub"`
	Resp RespPermission `json:"resp"`
}

type Permission struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

type RespPermission struct {
	Max int `json:"max"`
	TTL int `json:"ttl"`
}

type UserLimits struct {
	Max     int `json:"max"`
	Payload int `json:"payload"`

	// Src is a comma-separated list of CIDR blocks
	Src string `json:"src"`

	// Times is a list of start/end times in the format "15:04:05".
	Times []StartEndTime `json:"times"`
}

type StartEndTime struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	KeyPair    *KeyPair                 `json:"keyPair,omitempty"`
	AccountRef *InferredObjectReference `json:"accountRef,omitempty"`
	Conditions apis.Conditions          `json:"conditions,omitempty"`
}

func (s *UserStatus) GetConditions() apis.Conditions {
	return s.Conditions
}

func (s *UserStatus) SetConditions(conditions apis.Conditions) {
	s.Conditions = conditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
