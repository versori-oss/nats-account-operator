# NATS Account Operator Specification

This document describes the CRD specifications for the NATS Account Kubernetes Operator.

## CRD Types

### Operator

```yaml
apiVersion: accounts.nats.io/v1alpha1
kind: Operator
metadata:
  name: nats
  namespace: nats-io
spec:
  # The secret containing the operator's JWT in a file named nats.jwt
  jwtSecretName: nats-operator-jwt
  
  # The secret containing the operator's identity seed in a file named nats.seed
  seedSecretName: nats-operator-seed

  # Selector limiting which Namespaces Accounts may be defined in for this Operator. A null selector applies only to the 
  # current namespace.
  accountsNamespaceSelector: {}
  
  # Selector limiting which Accounts may be defined for this Operator. A null selector will allow all Accounts.
  accountsSelector: {}
  
  # All SigningKeys in the same namespace as the Operator matching the selector will be added to the set of signing keys
  # for the operator. Accounts created by this operator will be able to use these keys to sign their JWTs.
  signingKeysSelector: {}
  
  # The system account is a special account that can be used to access internal services exposed by NATS.
  systemAccountRef:
    name: sys
    namespace: "" # defaults to the current Namespace for this Operator resource.
  
  identities:
    - id: ""
      proof: ""
  accountServerURL: ""
  operatorServiceURLs: []
status:
  keyPair: {} # See KeyPair duck type below
  signingKeys:
    - name: ""
      keyPair: {} # See KeyPair duck type below
  systemAccountRef:
    name: ""
    namespace: ""
  conditions:
    - type: Ready
      status: "True"
    - type: SystemAccountResolved
      status: "True"
    - type: SystemAccountReady
      status: "True"
    - type: JWTSecretReady
      status: "True"
    - type: SeedSecretReady
      status: "True"
    - type: JWTPushed
      status: "True"
```

### Account

```yaml
apiVersion: accounts.nats.io/v1alpha1
kind: Account
metadata:
  name: sys
  namespace: nats-io
spec:
  signingKey:
    ref:
      # Any type implementing the "KeyPair" duck type, i.e. "Operator" or "SigningKey". The resultant resource must 
      # allow the Account based on its label and namespace selectors.
      apiVersion: accounts.nats.io/v1alpha1
      kind: SigningKey
      name: ""
      namespace: "" # empty namespace denotes the same namespace as this Account resource
  # Selector limiting which Namespaces Users may be defined in for this Account. A null selector applies only to the 
  # current namespace.
  usersNamespaceSelector: {}
  # Selector limiting which Users may be defined for this Account. A null or empty selector will allow all users
  usersSelector: {}
  # The secret containing the account's JWT in a file named nats.jwt
  jwtSecretName: nats-account-sys-jwt
  # The secret containing the account's identity seed in a file named nats.seed
  seedSecretName: nats-account-sys-seed
  # The selector limiting which SigningKeys may be used to sign JWTs for this Account. All SigningKeys must be in the 
  # same namespace as the Account.
  signingKeysSelector: {}
  imports:
    - name: ""
      subject: ""
      account: ""
      token: ""
      to: ""
      # Stream or Service
      type: ""
  exports: 
    - name: ""
      subject: ""
      # Stream or Service
      type: ""
      tokenReq: true
      # object of public key -> unix timestamp
      revocations: {}
      # Singleton, Stream or Chunked
      responseType: ""
      serviceLatency: 
        # range of 0-100
        sampling: 0
        results: ""
      accountTokenPosition: 0
  identities:
    - id: ""
      proof: ""
  limits:
    subs: -1
    conn: -1
    leaf: -1
    imports: -1
    exports: -1
    data: -1
    payload: -1
    wildcards: false
  # object of public key -> unix timestamp
  revocations: {}
status:
  keyPair: {} # See KeyPair duck type below
  signingKeys:
    - name: ""
      keyPair: {} # See KeyPair duck type below
  operatorRef:
    name: ""
    namespace: ""
  conditions:
    - type: Ready
      status: "True"
    - type: OperatorResolved
      status: "True"
    - type: SigningKeysUpToDate
      status: "True"
    - type: JWTSecretReady
      status: "True"
    - type: SeedSecretReady
      status: "True"
    - type: JWTPushed
      status: "True"
```

### User

```yaml
apiVersion: accounts.nats.io/v1alpha1
kind: User
metadata:
  name: sys
  namespace: nats-io
spec:
  signingKey:
    ref:
      # Any type implementing the "KeyPair" duck type, i.e. "Account" or "SigningKey". The resultant resource must 
      # allow the User based on its label and namespace selectors.
      apiVersion: accounts.nats.io/v1alpha1
      kind: SigningKey
      name: ""
      namespace: "" # empty namespace denotes the same namespace as this Account resource
  # The secret containing the account's JWT in a file named nats.jwt
  jwtSecretName: nats-account-sys-jwt
  # The secret containing the account's identity seed in a file named nats.seed
  seedSecretName: nats-account-sys-seed
  # The secret containing a decorated credential in a file named nats.creds
  credentialsSecretName: nats-account-sys-creds

  permissions:
    pub:
      allow: []
      deny: []
    sub:
      allow: []
      deny: []
    resp:
      max: -1
      ttl: -1
  limits:
    max: -1
    payload: -1
    # A comma-separated list of CIDR blocks
    src: ""
    # The start/end times in the format: 15:04:05
    times:
      - start: ""
        end: ""
  bearerToken: false
status:
  keyPair: {} # See KeyPair duck type below
  accountRef: 
    namespace: ""
    name: ""
  conditions:
    - type: Ready
      status: "True"
    - type: AccountResolved
      status: "True"
    - type: JWTSecretReady
      status: "True"
    - type: SeedSecretReady
      status: "True"
    - type: CredentialsSecretReady
      status: "True"
```

### SigningKey

```yaml
apiVersion: accounts.nats.io/v1alpha1
kind: SigningKey
metadata:
  name: sys-0
  namespace: nats-io
spec:
  # One of: Operator, Account, User. May be extended in future to allow for other types.
  type: "Account"
  # The secret containing the seed in a file named nats.seed
  seedSecretName: nats-account-sys-0-seed
status:
  keyPair: {} # See KeyPair duck type below
  ownerRef:
    apiVersion: accounts.nats.io/v1alpha1
    kind: Account
    name: sys
    namespace: nats-io
    uid: ""
  conditions:
    - type: Ready
      status: "True"
    - type: SeedSecretReady
      status: "True"
    - type: OwnerResolved
      status: "True"
```

## Duck types

In order to allow User/Account resources be signed by either their parent Operator/Account resource (or by a 
SigningKey owned by that parent) we must implement duck typing. This allows the controller to not have preexisting 
knowledge of a specific type ahead of time, and can instead read a resource based on its apiVersion and kind and know 
it will provide the information required to fulfil its purpose.

### KeyPair

The `KeyPair` duck type is implemented by all resources which have a public/private key pair stored on its status. The 
controller is responsible for ensuring the prefix matches the expected type based on its usage. For example, a User 
may only reference a KeyPair as its signing key if the prefix is `SA` for the seed and `A` for the public key.

```yaml
status:
  keyPair:
    publicKey: ""
    seedSecretName: ""
```

## Initial configuration

For users who already have their NKEY infrastructure established, you may pre-create the associated Secrets containing
JWTs, NKEY seeds and credentials. Upon reconciling a resource, if the Secrets already exists the controller will 
validate their contents and will simply update their statuses to reflect the current state. 

- Out-of-date JWTs will be updated to match the desired configuration
- Seeds will never be mutated.
- Missing Secrets will be created
- Failure to create a missing Secret or reading an existing secret will result in the resource being marked as 
  failed - see `.status.conditions` defined on each CRD type.
