package v1alpha1

import "github.com/versori-oss/nats-account-operator/pkg/apis"

const (
	UserConditionReady                  = apis.ConditionReady
	UserConditionAccountResolved        = "AccountResolved"
	UserConditionIssuerResolved         = "IssuerResolved"
	UserConditionJWTSecretReady         = "JWTSecretReady"
	UserConditionCredentialsSecretReady = "CredentialsSecretReady"
)

var userConditionSet = apis.NewLivingConditionSet(
	UserConditionReady,
	KeyPairableConditionSeedSecretReady,
	UserConditionAccountResolved,
	UserConditionIssuerResolved,
	UserConditionJWTSecretReady,
	UserConditionCredentialsSecretReady,
)

func (*User) GetConditionSet() apis.ConditionSet {
	return userConditionSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *UserStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return userConditionSet.Manage(s).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (s *UserStatus) IsReady() bool {
	return userConditionSet.Manage(s).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *UserStatus) InitializeConditions() {
	userConditionSet.Manage(s).InitializeConditions()
}

func (s *UserStatus) MarkAccountResolved(ref InferredObjectReference) {
	s.AccountRef = &ref

	userConditionSet.Manage(s).MarkTrue(UserConditionAccountResolved)
}

func (s *UserStatus) MarkAccountResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	s.AccountRef = nil

	userConditionSet.Manage(s).MarkFalse(UserConditionAccountResolved, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkAccountResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.AccountRef = nil

	userConditionSet.Manage(s).MarkUnknown(UserConditionAccountResolved, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkIssuerResolved() {
	userConditionSet.Manage(s).MarkTrue(UserConditionIssuerResolved)
}

func (s *UserStatus) MarkIssuerResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	userConditionSet.Manage(s).MarkFalse(UserConditionIssuerResolved, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkIssuerResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	userConditionSet.Manage(s).MarkUnknown(UserConditionIssuerResolved, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkJWTSecretReady() {
	userConditionSet.Manage(s).MarkTrue(UserConditionJWTSecretReady)
}

func (s *UserStatus) MarkJWTSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	userConditionSet.Manage(s).MarkFalse(UserConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkJWTSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	userConditionSet.Manage(s).MarkUnknown(UserConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkSeedSecretReady(kp KeyPair) {
	s.KeyPair = &kp

	userConditionSet.Manage(s).MarkTrue(KeyPairableConditionSeedSecretReady)
}

func (s *UserStatus) MarkSeedSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	userConditionSet.Manage(s).MarkFalse(KeyPairableConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkSeedSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	userConditionSet.Manage(s).MarkUnknown(KeyPairableConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkCredentialsSecretReady() {
	userConditionSet.Manage(s).MarkTrue(UserConditionCredentialsSecretReady)
}

func (s *UserStatus) MarkCredentialsSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	userConditionSet.Manage(s).MarkFalse(UserConditionCredentialsSecretReady, reason, messageFormat, messageA...)
}

func (s *UserStatus) MarkCredentialsSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	userConditionSet.Manage(s).MarkUnknown(UserConditionCredentialsSecretReady, reason, messageFormat, messageA...)
}
