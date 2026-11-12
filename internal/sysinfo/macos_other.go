//go:build !darwin

package sysinfo

// MacOSVersion is empty outside macOS.
func MacOSVersion() string { return "" }
