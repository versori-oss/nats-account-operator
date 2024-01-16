# Changelog

## [0.2.5](https://github.com/versori-oss/nats-account-operator/compare/v0.2.4...v0.2.5) (2024-01-16)


### Bug Fixes

* **rbac:** add missing group from manager-role ([d4163e8](https://github.com/versori-oss/nats-account-operator/commit/d4163e8b35aff9cdc49594830c75689851a8cdf8))

## [0.2.4](https://github.com/versori-oss/nats-account-operator/compare/v0.2.3...v0.2.4) (2024-01-16)


### Bug Fixes

* allow operator to create events in all namespaces and add shortNames for ease of use in k9s ([aa2cac0](https://github.com/versori-oss/nats-account-operator/commit/aa2cac062c31e7ea38c18e626404811dce8cb41e))
* handle difference between temporary and terminal errors ([20814a7](https://github.com/versori-oss/nats-account-operator/commit/20814a7ba93951793354efb37fc0b6a5279cacb1))

## [0.2.3](https://github.com/versori-oss/nats-account-operator/compare/v0.2.2...v0.2.3) (2023-11-07)


### Bug Fixes

* Don't compare pointer and struct because they are not equal ([0537e0c](https://github.com/versori-oss/nats-account-operator/commit/0537e0c9aa5b3916db79c262ca9a5d58b6d7a57a))
* update dependencies ([d6b3d8c](https://github.com/versori-oss/nats-account-operator/commit/d6b3d8c00208e3a64a7c0625ebf2515fafb3fce7))
* Update go version in CI ([b5acd29](https://github.com/versori-oss/nats-account-operator/commit/b5acd292f71ac905be761d368bca5d0ca508709f))
* user user condition set for user and not account set ([77879db](https://github.com/versori-oss/nats-account-operator/commit/77879db41faae3d534c3af9e588ac2c2b24620ac))

## [0.2.2](https://github.com/versori-oss/nats-account-operator/compare/v0.2.1...v0.2.2) (2023-08-08)


### Bug Fixes

* Push  all acounts JWT to nats ([6f10bd8](https://github.com/versori-oss/nats-account-operator/commit/6f10bd8b4138b0aadf79daf26afea5e08c7d3ad4))

## [0.2.1](https://github.com/versori-oss/nats-account-operator/compare/v0.2.0...v0.2.1) (2023-07-14)


### Bug Fixes

* use operator namespace when reading the TLS secret ([8b90bdb](https://github.com/versori-oss/nats-account-operator/commit/8b90bdb738b2e79cf7e3958bdfbda925134f2aaa))

## [0.2.0](https://github.com/versori-oss/nats-account-operator/compare/v0.1.2...v0.2.0) (2023-07-13)


### Features

* add ca.crt to the user creds secrets (conv. commit for CI) ([7f4f35c](https://github.com/versori-oss/nats-account-operator/commit/7f4f35c9bbbf0a5b518dddabd6d1211562a5a18c))

## [0.1.2](https://github.com/versori-oss/nats-account-operator/compare/v0.1.1...v0.1.2) (2023-07-12)


### Bug Fixes

* **owners:** only add owner references to resources created by the controller ([1dd9aa8](https://github.com/versori-oss/nats-account-operator/commit/1dd9aa883c66b3e2bf52b71f990cff133b3a4173))
* **tls-config:** NATS connections can be configured with a CA from a literal byte slice rather than a file ([34a5526](https://github.com/versori-oss/nats-account-operator/commit/34a55265172f18775c97f6b7f8f9391c093fd641))

## [0.1.1](https://github.com/versori-oss/nats-account-operator/compare/v0.1.0...v0.1.1) (2023-07-12)


### Features

* **operator:** ability to define CA certificate for connecting to NATS ([c0c06de](https://github.com/versori-oss/nats-account-operator/commit/c0c06de5ca81c0a0d9b6df10927a63acd1a44784))

## 0.1.0 (2023-07-11)


### Features

* add code-gen generated clientsets, informers and listers for CRDs ([e87d39f](https://github.com/versori-oss/nats-account-operator/commit/e87d39f05d154de238c710f5975d92c7b3759801))
* big changes, things seem to work now, docs added on how to test ([690b23d](https://github.com/versori-oss/nats-account-operator/commit/690b23de47535456a85736e60e1890f8b8ea4d88))
* initial commit with README.md and ./docs/specification.md ([82f2c27](https://github.com/versori-oss/nats-account-operator/commit/82f2c27abaa9d87ae9ab4ad8338c507649289ea1))


### Bug Fixes

* **accounts-controller:** ensure regular account actually proceeds into pushing JWTs ([49268cd](https://github.com/versori-oss/nats-account-operator/commit/49268cd08cbc842604906374c4b2c7d3692a10e9))
* **accounts-controller:** fixed a wrong error check causing resources not to become ready when they should be ([38d521a](https://github.com/versori-oss/nats-account-operator/commit/38d521ad51685f1d45ac87f2c909e3003856dbbe))
* **conversion:** enabled import/export types to be either capitalized or not (Stream/stream) and fixed a panic caused by not checking optional service latency parameter in exports ([50e2239](https://github.com/versori-oss/nats-account-operator/commit/50e2239a3ce53f3fe81e63430969b9f733477efa))
* correctly resolve SigningKey owner references ([9a2066e](https://github.com/versori-oss/nats-account-operator/commit/9a2066e1e2ad48ef37f9ec0239d405a662b8682c))
* **creds:** ensure that account and user claims are validated ([1c1ac85](https://github.com/versori-oss/nats-account-operator/commit/1c1ac85f3c033dc12f0718ae63a90f62e45a2872))
* enable accounts to be used as signingKey's in user specs ([c280a51](https://github.com/versori-oss/nats-account-operator/commit/c280a519c75429702e818597f4121ca6b0fedf59))
* error handling, logging and status writing ([ca8b57e](https://github.com/versori-oss/nats-account-operator/commit/ca8b57e55902348a258d9db85c6968d2c3c828d3))
* fixed panics caused by resources not being ready and found the cause of the cascading reconcile requests. ([3013473](https://github.com/versori-oss/nats-account-operator/commit/301347309c041c69fec40822d3d68c1c5ec647a6))
* fixed the infinite reconcile cascade by not updating jwt secrets every iteration plus refactoring ([1c71024](https://github.com/versori-oss/nats-account-operator/commit/1c71024783972da5fedf184429343806929a8962))
* handle respone types in accountexports better and add comments in struct to aid users to put correct values in ([2e9cbb1](https://github.com/versori-oss/nats-account-operator/commit/2e9cbb1e015f6da550244370acc99d8ed9b627da))
* **lifecycles:** use correct condition sets for user/signingkey/account lifecycle methods ([96d8305](https://github.com/versori-oss/nats-account-operator/commit/96d83059327a521632ff5aa484677d0a54927728))
* make sure the system account has a user with credentials to log in with. Log into the NATS server with these user credentials rather than account credentials ([57e4866](https://github.com/versori-oss/nats-account-operator/commit/57e4866958b9046234269e0367da1c3232acd6c8))
* **nats-jwt:** upgrade jwt package to v2 and update vendor directory ([9cc317a](https://github.com/versori-oss/nats-account-operator/commit/9cc317a6381410319d6365acb1fee9e83bbee429))
* **nats:** added nsc package. Now create a new nats client everytime a JWT needs to be pushed/updated. ([6352082](https://github.com/versori-oss/nats-account-operator/commit/6352082b529152e8bc140ee49b8be0ab1010a5ba))
* prevent system account jwts from being pushed to nats ([5e6f131](https://github.com/versori-oss/nats-account-operator/commit/5e6f1312cf879792abdbf243da509d72df76c10b))
* rbac rules and tidy up manifests ([125f738](https://github.com/versori-oss/nats-account-operator/commit/125f7385dd98aa172e52b80c0abaad7974d0905c))
* tested and fixed issues with account and operator controller ([f6dba16](https://github.com/versori-oss/nats-account-operator/commit/f6dba169df4034c23d68d1f51590262d2cb82c4f))
* update code to use v2 of nats jwt package from v1 ([46acffa](https://github.com/versori-oss/nats-account-operator/commit/46acffae783d4910b7a59958c64084b87118dd4b))
* user jwt's need to have the accounts public key as their issuer_account field in order to be able to log in. ([55fa787](https://github.com/versori-oss/nats-account-operator/commit/55fa787129eec5622bc6ac035f27a8d6e8a48d86))
