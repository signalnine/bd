//go:build windows

package config

import (
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

// CheckBdDirPermissions is a no-op on Windows where filesystem
// permissions use ACLs rather than Unix permission bits.
func CheckBdDirPermissions(path string) {}
