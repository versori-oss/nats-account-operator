package resources

const (
	LabelSecretType            = "nats.accounts.io/secret-type"
	LabelSecretTypeSeed        = "seed"
	LabelSecretTypeJWT         = "jwt"
	LabelSecretTypeCredentials = "credentials"

	LabelSecretJWTType  = "nats.accounts.io/jwt-type"
	LabelSecretSeedType = "nats.accounts.io/seed-type"

	LabelSecretTypeOperator   = "Operator"
	LabelSecretTypeSigningKey = "SigningKey"
	LabelSecretTypeAccount    = "Account"
	LabelSecretTypeUser       = "User"

	LabelSubject = "accounts.nats.io/subject"

	LabelOperatorName   = "accounts.nats.io/operator"
	LabelSigningKeyName = "accounts.nats.io/signing-key"
	LabelAccountName    = "accounts.nats.io/account"
	LabelUserName       = "accounts.nats.io/user"
)
