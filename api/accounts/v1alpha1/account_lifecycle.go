package v1alpha1

import "github.com/versori-oss/nats-account-operator/pkg/apis"

const (
	AccountConditionReady              = apis.ConditionReady
	AccountConditionOperatorResolved   = "OperatorResolved"
	AccountConditionSigningKeysUpdated = "SigningKeysUpdated"
	AccountConditionJWTSecretReady     = "JWTSecretReady"
	AccountConditionSeedSecretReady    = "SeedSecretReady"
	AccountConditionJWTPushed          = "JWTPushed"
)

var accountConditionSet = apis.NewLivingConditionSet(
	AccountConditionReady,
	AccountConditionOperatorResolved,
	AccountConditionSigningKeysUpdated,
	AccountConditionJWTSecretReady,
	AccountConditionSeedSecretReady,
	AccountConditionJWTPushed,
)

func (*Account) GetConditionSet() apis.ConditionSet {
	return accountConditionSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *AccountStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return operatorConditionSet.Manage(s).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (s *AccountStatus) IsReady() bool {
	return operatorConditionSet.Manage(s).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *AccountStatus) InitializeConditions() {
	operatorConditionSet.Manage(s).InitializeConditions()
}

func (s *AccountStatus) MarkOperatorResolved(ref InferredObjectReference) {
	s.OperatorRef = &ref

	operatorConditionSet.Manage(s).MarkTrue(AccountConditionOperatorResolved)
}

func (s *AccountStatus) MarkOperatorResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	s.OperatorRef = nil

	operatorConditionSet.Manage(s).MarkFalse(AccountConditionOperatorResolved, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkOperatorResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.OperatorRef = nil

	operatorConditionSet.Manage(s).MarkUnknown(AccountConditionOperatorResolved, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSigningKeysUpdated(signingKeys []SigningKeyEmbeddedStatus) {
	s.SigningKeys = signingKeys

	operatorConditionSet.Manage(s).MarkTrueWithReason(OperatorConditionSigningKeysUpdated, OperatorConditionSigningKeysUpdated, "Found %d signing keys", len(signingKeys))
}

func (s *AccountStatus) MarkSigningKeysUpdateFailed(reason, messageFormat string, messageA ...interface{}) {
	s.SigningKeys = nil

	operatorConditionSet.Manage(s).MarkFalse(OperatorConditionSigningKeysUpdated, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSigningKeysUpdateUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.SigningKeys = nil

	operatorConditionSet.Manage(s).MarkUnknown(OperatorConditionSigningKeysUpdated, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTSecretReady() {
	operatorConditionSet.Manage(s).MarkTrue(AccountConditionJWTSecretReady)
}

func (s *AccountStatus) MarkJWTSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(s).MarkFalse(AccountConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(s).MarkUnknown(AccountConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSeedSecretReady(publicKey, seedSecretName string) {
	s.KeyPair = &KeyPair{
		PublicKey:      publicKey,
		SeedSecretName: seedSecretName,
	}

	operatorConditionSet.Manage(s).MarkTrue(AccountConditionSeedSecretReady)
}

func (s *AccountStatus) MarkSeedSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	operatorConditionSet.Manage(s).MarkFalse(AccountConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSeedSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	operatorConditionSet.Manage(s).MarkUnknown(AccountConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTPushed() {
	operatorConditionSet.Manage(s).MarkTrue(AccountConditionJWTPushed)
}

func (s *AccountStatus) MarkJWTPushFailed(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(s).MarkFalse(AccountConditionJWTPushed, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTPushUnknown(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(s).MarkUnknown(AccountConditionJWTPushed, reason, messageFormat, messageA...)
}
