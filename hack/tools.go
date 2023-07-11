//go:build tools
// +build tools

package tools

import (
	_ "github.com/vektra/mockery/v2"
	_ "k8s.io/client-go"
	_ "k8s.io/code-generator"
)
