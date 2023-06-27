package v1alpha1

import "github.com/versori-oss/nats-account-operator/pkg/apis"

const (
	OperatorConditionReady                 = apis.ConditionReady
	OperatorConditionSystemAccountResolved = "SystemAccountResolved"
	OperatorConditionSystemAccountReady    = "SystemAccountReady"
	OperatorConditionSigningKeysUpdated    = "SigningKeysUpdated"
	OperatorConditionJWTSecretReady        = "JWTSecretReady"
	OperatorConditionSeedSecretReady       = "SeedSecretReady"
)

var operatorConditionSet = apis.NewLivingConditionSet(
	OperatorConditionReady,
	OperatorConditionSystemAccountResolved,
	OperatorConditionSystemAccountReady,
	OperatorConditionSigningKeysUpdated,
	OperatorConditionJWTSecretReady,
	OperatorConditionSeedSecretReady,
)

func (*Operator) GetConditionSet() apis.ConditionSet {
	return operatorConditionSet
}

func (o *Operator) GetConditionManager() apis.ConditionManager {
	return operatorConditionSet.Manage(&o.Status)
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (os *OperatorStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return operatorConditionSet.Manage(os).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (os *OperatorStatus) IsReady() bool {
	return operatorConditionSet.Manage(os).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (os *OperatorStatus) InitializeConditions() {
	operatorConditionSet.Manage(os).InitializeConditions()
}

func (os *OperatorStatus) MarkSystemAccountResolved(ref InferredObjectReference) {
	os.ResolvedSystemAccount = &ref

	operatorConditionSet.Manage(os).MarkTrue(OperatorConditionSystemAccountResolved)
}

func (os *OperatorStatus) MarkSystemAccountResolveFailed(reason, messageFormat string, messageA ...interface{}) {
	os.ResolvedSystemAccount = nil

	operatorConditionSet.Manage(os).MarkFalse(OperatorConditionSystemAccountResolved, reason, messageFormat, messageA...)

	os.MarkSystemAccountUnknown(reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSystemAccountResolveUnknown(reason, messageFormat string, messageA ...interface{}) {
	os.ResolvedSystemAccount = nil

	operatorConditionSet.Manage(os).MarkUnknown(OperatorConditionSystemAccountResolved, reason, messageFormat, messageA...)

	os.MarkSystemAccountUnknown(reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSystemAccountReady() {
	operatorConditionSet.Manage(os).MarkTrue(OperatorConditionSystemAccountReady)
}

func (os *OperatorStatus) MarkSystemAccountNotReady(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(os).MarkFalse(OperatorConditionSystemAccountReady, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSystemAccountUnknown(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(os).MarkUnknown(OperatorConditionSystemAccountReady, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSigningKeysUpdated(signingKeys []SigningKeyEmbeddedStatus) {
	os.SigningKeys = signingKeys

	operatorConditionSet.Manage(os).MarkTrueWithReason(OperatorConditionSigningKeysUpdated, OperatorConditionSigningKeysUpdated, "Found %d signing keys", len(signingKeys))
}

func (os *OperatorStatus) MarkSigningKeysUpdateFailed(reason, messageFormat string, messageA ...interface{}) {
	os.SigningKeys = nil

	operatorConditionSet.Manage(os).MarkFalse(OperatorConditionSigningKeysUpdated, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSigningKeysUpdateUnknown(reason, messageFormat string, messageA ...interface{}) {
	os.SigningKeys = nil

	operatorConditionSet.Manage(os).MarkUnknown(OperatorConditionSigningKeysUpdated, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkJWTSecretReady() {
	operatorConditionSet.Manage(os).MarkTrue(OperatorConditionJWTSecretReady)
}

func (os *OperatorStatus) MarkJWTSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(os).MarkFalse(OperatorConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkJWTSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	operatorConditionSet.Manage(os).MarkUnknown(OperatorConditionJWTSecretReady, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSeedSecretReady(publicKey, seedSecretName string) {
	os.KeyPair = &KeyPair{
		PublicKey:      publicKey,
		SeedSecretName: seedSecretName,
	}

	operatorConditionSet.Manage(os).MarkTrue(OperatorConditionSeedSecretReady)
}

func (os *OperatorStatus) MarkSeedSecretFailed(reason, messageFormat string, messageA ...interface{}) {
	os.KeyPair = nil

	operatorConditionSet.Manage(os).MarkFalse(OperatorConditionSeedSecretReady, reason, messageFormat, messageA...)
}

func (os *OperatorStatus) MarkSeedSecretUnknown(reason, messageFormat string, messageA ...interface{}) {
	os.KeyPair = nil

	operatorConditionSet.Manage(os).MarkUnknown(OperatorConditionSeedSecretReady, reason, messageFormat, messageA...)
}
