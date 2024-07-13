package internal

import (
	"errors"
	"os"
)

func CheckIfFileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		// Schrödinger: file may or may not exist. See err for details.
		panic(err)
	}
}
