// Package errors contains the domain errors of Kamino.
package errors

// Kamino is implemented by every domain error.
type Kamino interface {
	error
	kamino()
}

// KaminoError is the base of the domain errors.
type KaminoError struct{ Msg string }

func (e KaminoError) Error() string { return e.Msg }
func (KaminoError) kamino()         {}

// OperationCancelled means the user cancelled a cooperative operation.
type OperationCancelled struct{ KaminoError }

// CatalogError means an application definition is invalid or unreadable.
type CatalogError struct{ KaminoError }

// ConfigError means the TUI configuration is invalid or unreadable.
type ConfigError struct{ KaminoError }

// DetectionError means an application could not be inspected.
type DetectionError struct{ KaminoError }

// ArchiveError means a clone archive could not be read or written.
type ArchiveError struct{ KaminoError }

// IntegrityError means a checksum or archive member does not match.
type IntegrityError struct{ KaminoError }

// VersionError means the archive format version is not supported.
type VersionError struct{ KaminoError }

// PathError means a path escapes the allowed home or archive scope.
type PathError struct{ KaminoError }

// UpdateErrorKind classifies an update failure.
type UpdateErrorKind string

const (
	UpdateNotSelfContained UpdateErrorKind = "not_self_contained"
	UpdatePlatform         UpdateErrorKind = "platform"
	UpdateNetwork          UpdateErrorKind = "network"
	UpdateIntegrity        UpdateErrorKind = "integrity"
	UpdateReplace          UpdateErrorKind = "replace"
	UpdateOther            UpdateErrorKind = "other"
)

// UpdateError means the self-update could not complete.
type UpdateError struct {
	KaminoError
	Kind UpdateErrorKind
}

// ConflictError means deploy conflicts cannot be resolved as requested.
type ConflictError struct{ KaminoError }

// DecisionRequiredError means a conflict has no decision.
type DecisionRequiredError struct{ KaminoError }

// InvalidMergeError means a composed file is invalid while both originals were valid.
type InvalidMergeError struct{ KaminoError }

// ConcurrentChangeError means a destination changed between planning and applying.
type ConcurrentChangeError struct{ KaminoError }

// PartialApplyError means some writes were applied and a later one failed.
type PartialApplyError struct {
	KaminoError
	Applied  []string
	Failed   string
	Snapshot string
}

// AsKamino reports whether err is a Kamino domain error.
func AsKamino(err error) (Kamino, bool) {
	k, ok := err.(Kamino)
	return k, ok
}
