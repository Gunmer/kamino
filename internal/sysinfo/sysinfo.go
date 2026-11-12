// Package sysinfo gathers the metadata stored in clones.
package sysinfo

import (
	"os"
	"runtime"
	"time"
)

// Arch returns the machine architecture with the platform naming used by Python.
func Arch() string { return runtime.GOARCH }

// Hostname returns the machine hostname.
func Hostname() string {
	value, err := os.Hostname()
	if err != nil {
		return ""
	}
	return value
}

// NowISO returns the current UTC time in ISO 8601 with a Z suffix.
func NowISO() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }
