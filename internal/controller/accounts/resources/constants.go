package resources

const (
	AnnotationSecretType            = "nats.accounts.io/secret-type"
	AnnotationSecretTypeSeed        = "seed"
	AnnotationSecretTypeJWT         = "jwt"
	AnnotationSecretTypeCredentials = "credentials"

	AnnotationSecretJWTType  = "nats.accounts.io/jwt-type"
	AnnotationSecretSeedType = "nats.accounts.io/seed-type"

	AnnotationSecretTypeOperator   = "Operator"
	AnnotationSecretTypeSigningKey = "SigningKey"
	AnnotationSecretTypeAccount    = "Account"
	AnnotationSecretTypeUser       = "User"

	LabelSubject = "accounts.nats.io/subject"

	LabelOperatorName   = "accounts.nats.io/operator"
	LabelSigningKeyName = "accounts.nats.io/signing-key"
	LabelAccountName    = "accounts.nats.io/account"
	LabelUserName       = "accounts.nats.io/user"
)
