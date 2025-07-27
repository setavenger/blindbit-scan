package internal

import (
	"errors"
	"os"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/utils"
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

type Status string

const (
	StatusSetupRequired Status = "setup-required"
	StatusLocked        Status = "locked"
	StatusRunning       Status = "running"
)

func InstanceStatus() Status {
	if !utils.CheckIfFileExists(config.PathDbAuth) {
		// wallet is not setup yet
		return StatusSetupRequired
	}
	creds := config.GetAuthCredentials()
	if creds == nil {
		// wallet is locked
		return StatusLocked
	}

	return StatusRunning
}
