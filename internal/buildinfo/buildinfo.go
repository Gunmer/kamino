// Package buildinfo exposes build-time flags.
package buildinfo

// SelfContained is set to "true" by release builds with -X.
var SelfContained = "false"

// IsSelfContained reports whether the running binary is a release binary.
func IsSelfContained() bool { return SelfContained == "true" }
