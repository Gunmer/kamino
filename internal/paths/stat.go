package paths

import (
	"os"

	"golang.org/x/sys/unix"
)

// KindDir is the classification for a directory discovered by Walk.
const KindDir Kind = "dir"

func infoFromStat(st *unix.Stat_t) Info {
	mode := uint32(st.Mode)
	kind := KindSpecial
	switch mode & unix.S_IFMT {
	case unix.S_IFREG:
		kind = KindFile
	case unix.S_IFDIR:
		kind = KindDir
	case unix.S_IFLNK:
		kind = KindLink
	}
	return Info{
		Kind:  kind,
		Perm:  os.FileMode(mode & 0o7777),
		Size:  st.Size,
		UID:   st.Uid,
		GID:   st.Gid,
		Mtime: st.Mtim.Sec,
		Ino:   st.Ino,
	}
}
