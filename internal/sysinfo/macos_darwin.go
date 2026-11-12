//go:build darwin

package sysinfo

import "golang.org/x/sys/unix"

// MacOSVersion returns the running macOS product version.
func MacOSVersion() string {
	value, err := unix.Sysctl("kern.osproductversion")
	if err != nil {
		return ""
	}
	return value
}
