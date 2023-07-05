package v1alpha1

import "github.com/versori-oss/nats-account-operator/pkg/apis"

const (
	AccountConditionReady              = apis.ConditionReady
	AccountConditionOperatorResolved   = "OperatorResolved"
	AccountConditionIssuerResolved     = "IssuerResolved"
	AccountConditionSigningKeysUpdated = "SigningKeysUpdated"
	AccountConditionJWTSecretReady     = "JWTSecretReady"
	AccountConditionJWTPushed          = "JWTPushed"
)

var accountConditionSet = apis.NewLivingConditionSet(
	AccountConditionReady,
	KeyPairableConditionSeedSecretReady,
	AccountConditionOperatorResolved,
	AccountConditionIssuerResolved,
	AccountConditionSigningKeysUpdated,
	AccountConditionJWTSecretReady,
	AccountConditionJWTPushed,
)

func (*Account) GetConditionSet() apis.ConditionSet {
	return accountConditionSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *AccountStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return accountConditionSet.Manage(s).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (s *AccountStatus) IsReady() bool {
	return accountConditionSet.Manage(s).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *AccountStatus) InitializeConditions() {
	accountConditionSet.Manage(s).InitializeConditions()
}

func (s *AccountStatus) MarkOperatorResolved(ref InferredObjectReference) {
	s.OperatorRef = &ref

	accountConditionSet.Manage(s).MarkTrue(AccountConditionOperatorResolved)
}

func (s *AccountStatus) MarkOperatorResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	s.OperatorRef = nil

	accountConditionSet.Manage(s).MarkFalse(AccountConditionOperatorResolved, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkOperatorResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.OperatorRef = nil

	accountConditionSet.Manage(s).MarkUnknown(AccountConditionOperatorResolved, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkIssuerResolved() {
	accountConditionSet.Manage(s).MarkTrue(AccountConditionIssuerResolved)
}

func (s *AccountStatus) MarkIssuerResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	accountConditionSet.Manage(s).MarkFalse(AccountConditionIssuerResolved, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkIssuerResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	accountConditionSet.Manage(s).MarkUnknown(AccountConditionIssuerResolved, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSigningKeysUpdated(signingKeys []SigningKeyEmbeddedStatus) {
	s.SigningKeys = signingKeys

	accountConditionSet.Manage(s).MarkTrueWithReason(AccountConditionSigningKeysUpdated, "Signing keys updated", "Found %d signing keys", len(signingKeys))
}

func (s *AccountStatus) MarkSigningKeysUpdateFailed(reason, messageFormat string, messageA ...interface{}) {
	s.SigningKeys = nil

	accountConditionSet.Manage(s).MarkFalse(AccountConditionSigningKeysUpdated, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSigningKeysUpdateUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.SigningKeys = nil

	accountConditionSet.Manage(s).MarkUnknown(AccountConditionSigningKeysUpdated, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTSecretReady() {
	accountConditionSet.Manage(s).MarkTrue(AccountConditionJWTSecretReady)
}

func (s *AccountStatus) MarkJWTSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	accountConditionSet.Manage(s).MarkFalse(AccountConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	accountConditionSet.Manage(s).MarkUnknown(AccountConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSeedSecretReady(publicKey, seedSecretName string) {
	s.KeyPair = &KeyPair{
		PublicKey:      publicKey,
		SeedSecretName: seedSecretName,
	}

	accountConditionSet.Manage(s).MarkTrue(KeyPairableConditionSeedSecretReady)
}

func (s *AccountStatus) MarkSeedSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	accountConditionSet.Manage(s).MarkFalse(KeyPairableConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkSeedSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	s.KeyPair = nil

	accountConditionSet.Manage(s).MarkUnknown(KeyPairableConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTPushed() {
	accountConditionSet.Manage(s).MarkTrue(AccountConditionJWTPushed)
}

func (s *AccountStatus) MarkJWTPushFailed(reason, messageFormat string, messageA ...interface{}) {
	accountConditionSet.Manage(s).MarkFalse(AccountConditionJWTPushed, reason, messageFormat, messageA...)
}

func (s *AccountStatus) MarkJWTPushUnknown(reason, messageFormat string, messageA ...interface{}) {
	accountConditionSet.Manage(s).MarkUnknown(AccountConditionJWTPushed, reason, messageFormat, messageA...)
}
