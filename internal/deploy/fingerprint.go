package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"

	"github.com/Gunmer/kamino/internal/paths"
)

type fingerprint struct {
	kind   string
	sha256 string
	link   string
}

type resolvedFile struct {
	path        string
	destination string
	content     []byte
	link        string
	mode        string
	category    ResultCategory
	write       bool
	localText   *string
	cloneText   *string
	mergedText  *string
}

// fingerprintOf summarizes the destination. A missing entry is a valid state;
// any other failure is returned so an unreadable file is never mistaken for an
// absent one.
func fingerprintOf(safe *paths.SafeHome, rel string) (fingerprint, error) {
	info, err := safe.Lstat(rel)
	if errors.Is(err, fs.ErrNotExist) {
		return fingerprint{kind: "missing"}, nil
	}
	if err != nil {
		return fingerprint{}, err
	}
	switch info.Kind {
	case paths.KindLink:
		target, err := safe.Readlink(rel)
		if err != nil {
			return fingerprint{}, err
		}
		return fingerprint{kind: "symlink", link: target}, nil
	case paths.KindFile:
		data, err := safe.ReadBytes(rel)
		if err != nil {
			return fingerprint{}, err
		}
		digest := sha256.Sum256(data)
		return fingerprint{kind: "file", sha256: hex.EncodeToString(digest[:])}, nil
	default:
		return fingerprint{kind: "special"}, nil
	}
}
