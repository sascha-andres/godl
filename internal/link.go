//go:build !windows

package internal

import (
	"path"
	"strings"
)

// CreateSymlinkPath returns simply the place where the symbolic link will be created
func CreateSymlinkPath(destinationDir, symbolicLinkname string) string {
	if strings.HasPrefix(symbolicLinkname, "/") {
		return symbolicLinkname
	}
	return path.Join(destinationDir, symbolicLinkname)
}

// Link will create a symbolic link from src to dst
func Link(src, dst string) error {
	if _, err := os.Stat(dst); errors.Is(err, fs.ErrNotExist) {
		return linkInternal(src, dst)
	}

	fi, err := os.Lstat(dst)
	if err != nil {
		return fmt.Errorf("could not check if destination is a symbolic link: %w", err)
	}

	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		err = os.Remove(dst)
		if err != nil {
			return fmt.Errorf("could not remove old symbolic link: %w", err)
		}
	}

	return linkInternal(src, dst)
}

func linkInternal(src, dst string) error {
	return os.Symlink(src, dst)
}
