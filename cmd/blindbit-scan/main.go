package main

import (
	"os"
	"os/signal"

	"github.com/setavenger/blindbit-scan/internal/startup"
)

// func init() {
// 	// todo can this double reference work?
// 	flag.StringVar(
// 		&config.DirectoryPath,
// 		"datadir",
// 		config.DefaultDirectoryPath,
// 		"Set the base directory for blindbit-scan. Default directory is ~/.blindbit-scan.",
// 	)

// 	flag.BoolVar(&config.PrivateMode, "private", false, "BlindBit Scan will run in private mode. All data on disk will be encrypted all data will only be decrypted in memory. Upon restart the unlock endpoint needs to be called to decrypt data and start the scanning.")

// 	flag.Parse()
// }

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	startup.RunProgram()

	// wait for program stop signal
	<-interrupt
}
