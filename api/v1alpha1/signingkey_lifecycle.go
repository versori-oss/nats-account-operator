package v1alpha1

import "github.com/versori-oss/nats-account-operator/pkg/apis"

const (
	SigningKeyConditionReady           = apis.ConditionReady
	SigningKeyConditionSeedSecretReady = "SeedSecretReady"
	SigningKeyConditionOwnerResolved   = "OwnerResolved"
)

var signingKeyConditionSet = apis.NewLivingConditionSet(
	SigningKeyConditionReady,
	SigningKeyConditionSeedSecretReady,
	SigningKeyConditionOwnerResolved,
)

func (*SigningKey) GetConditionSet() apis.ConditionSet {
	return signingKeyConditionSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *SigningKeyStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return operatorConditionSet.Manage(s).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (s *SigningKeyStatus) IsReady() bool {
	return operatorConditionSet.Manage(s).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *SigningKeyStatus) InitializeConditions() {
	operatorConditionSet.Manage(s).InitializeConditions()
}

func (s *SigningKeyStatus) MarkSeedSecretReady(publicKey, seedSecretName string) {
	s.KeyPair = &KeyPair{
		PublicKey:      publicKey,
		SeedSecretName: seedSecretName,
	}

	operatorConditionSet.Manage(s).MarkTrue(SigningKeyConditionSeedSecretReady)
}

func (s *SigningKeyStatus) MarkSeedSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	operatorConditionSet.Manage(s).MarkFalse(SigningKeyConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *SigningKeyStatus) MarkSeedSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	operatorConditionSet.Manage(s).MarkUnknown(SigningKeyConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *SigningKeyStatus) MarkOwnerResolved(ref TypedObjectReference) {
	s.OwnerRef = &ref

	operatorConditionSet.Manage(s).MarkTrue(SigningKeyConditionOwnerResolved)
}

func (s *SigningKeyStatus) MarkOwnerResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	s.OwnerRef = nil

	operatorConditionSet.Manage(s).MarkFalse(SigningKeyConditionOwnerResolved, reason, messageFormat, messageA...)
}

func (s *SigningKeyStatus) MarkOwnerResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.OwnerRef = nil

	operatorConditionSet.Manage(s).MarkUnknown(SigningKeyConditionOwnerResolved, reason, messageFormat, messageA...)
}
