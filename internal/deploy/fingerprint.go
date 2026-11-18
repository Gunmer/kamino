package deploy

import (
	"crypto/sha256"
	"encoding/hex"

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

func fingerprintOf(safe *paths.SafeHome, rel string) fingerprint {
	info, err := safe.Lstat(rel)
	if err != nil {
		return fingerprint{kind: "missing"}
	}
	switch info.Kind {
	case paths.KindLink:
		target, err := safe.Readlink(rel)
		if err != nil {
			return fingerprint{kind: "special"}
		}
		return fingerprint{kind: "symlink", link: target}
	case paths.KindFile:
		data, err := safe.ReadBytes(rel)
		if err != nil {
			return fingerprint{kind: "special"}
		}
		digest := sha256.Sum256(data)
		return fingerprint{kind: "file", sha256: hex.EncodeToString(digest[:])}
	default:
		return fingerprint{kind: "special"}
	}
}
