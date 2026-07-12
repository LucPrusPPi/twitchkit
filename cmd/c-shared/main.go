//go:build cgo

// Command c-shared builds the C ABI shared library (requires a C toolchain).
//
//	CGO_ENABLED=1 go build -buildmode=c-shared -o twitchkit.dll ./cmd/c-shared
package main

import _ "github.com/LucPrusPPi/twitchkit/capi"

func main() {}
