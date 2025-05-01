package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/setavenger/blindbit-scan/pkg/logging"
)

// ResolvePath resolves a path, expanding ~ to the user's home directory
func ResolvePath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			logging.L.Panic().Err(err).Msg("failed to get user home directory")
		}
		path = filepath.Join(home, path[1:])
	}
	return path
}

// CheckIfFileExists checks if a file exists
func CheckIfFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TryCreateDirectoryPanic creates a directory and panics if it fails
func TryCreateDirectoryPanic(path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to create directory")
	}
}
