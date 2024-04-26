package v1alpha1

const (
	ReasonUnsupportedIssuer        = "UnsupportedIssuer"
	ReasonInvalidSigningKeyOwner   = "InvalidSigningKeyOwner"
	ReasonNotReady                 = "NotReady"
	ReasonNotFound                 = "NotFound"
	ReasonNotAllowed               = "NotAllowed"
	ReasonUnknownError             = "UnknownError"
	ReasonMalformedSeedSecret      = "MalformedSeedSecret"
	ReasonIssuerSeedError          = "IssuerSeedError"
	ReasonPublicKeyMismatch        = "PublicKeyMismatch"
	ReasonInvalidSeedSecret        = "InvalidSeedSecret"
	ReasonInvalidJWTSecret         = "InvalidJWTSecret"
	ReasonInvalidCredentialsSecret = "InvalidCredentialsSecret"
	ReasonJWTPushError             = "JWTPushError"
)
