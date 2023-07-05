package nsc

import (
    "github.com/nats-io/jwt/v2"
    "k8s.io/apimachinery/pkg/conversion"
)

// Equality provides DeepEqual capabilities for jwt related types, taking into account that some fields will always
// change between reconciles such as JWT IDs and IssuedAt timestamps.
var Equality = conversion.EqualitiesOrDie(
    // we have a copy of the jwt.ClaimsData type, so we can set non-comparable fields to their zero values then do a
    // straight forward == comparison.
	func(a, b jwt.ClaimsData) bool {
        a.ID = ""
        b.ID = ""
        a.IssuedAt = 0
        b.IssuedAt = 0

        return a == b
    },
)
