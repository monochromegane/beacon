package storage

import (
	"os"
	"path/filepath"
)

// ResolveBaseDir returns the base directory for beacon cache files.
func ResolveBaseDir() (string, error) {
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "beacon"), nil
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCache, "beacon"), nil
}
