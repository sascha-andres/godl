//go:build windows

package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CreateSymlinkPath returns simply the place where the symbolic link will be created
func CreateSymlinkPath(destinationDir, symbolicLinkname string) string {
	return path.Join(destinationDir, symbolicLinkname)
}

// Link will create a symbolic link from src to dst
func Link(src, dst string) error {
	if _, err := os.Stat(dst); errors.Is(err, fs.ErrNotExist) {
		return copyDirectory(src, dst)
	}

	err := os.RemoveAll(dst)
	if err != nil {
		return fmt.Errorf("could not remove destination: %w", err)
	}

	return copyDirectory(src, dst)
}

// copyDirectory will just copy the entire dir structure, expects the target to not exist (should
// be ensured by Link)
func copyDirectory(source, destination string) error {
	var err error
	err = os.MkdirAll(destination, 0770)
	if err != nil {
		return fmt.Errorf("could not create destination: %w", err)
	}

	src := strings.Replace(source, "\\", "/", -1)
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		var relPath = strings.Replace(path, src, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath[len(source):]), 0755)
		} else {
			var data, err1 = os.ReadFile(relPath)
			if err1 != nil {
				return err1
			}
			return os.WriteFile(filepath.Join(destination, relPath[len(source):]), data, 0777)
		}
	})
	return err
}
