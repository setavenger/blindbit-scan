package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var (
	L zerolog.Logger
)

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	L = zerolog.New(output).With().Caller().Timestamp().Logger()
}
