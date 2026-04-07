//go:build !windows

package config

import (
	"fmt"
	"io/fs"
	"os"
)

const (
	// BdDirPerm is the permission mode for .bd/ directories (owner-only).
	BdDirPerm fs.FileMode = 0700
	// BdFilePerm is the permission mode for state files inside .bd/ (owner-only).
	BdFilePerm fs.FileMode = 0600
)

// EnsureBdDir creates the .bd directory with secure permissions.
func EnsureBdDir(path string) error {
	return os.MkdirAll(path, BdDirPerm)
}

// CheckBdDirPermissions warns to stderr if the .bd directory has
// group or world-accessible permissions. The check is non-fatal.
func CheckBdDirPermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return // directory doesn't exist yet
	}
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: %s has permissions %04o (recommended: 0700). Run: chmod 700 %s\n", path, perm, path)
	}
}
