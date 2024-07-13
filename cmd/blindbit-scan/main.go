package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	"github.com/setavenger/blindbit-scan/internal/server"
)

func init() {
	// todo can this double reference work?
	flag.StringVar(&config.DirectoryPath, "datadir", config.DefaultDirectoryPath, "Set the base directory for blindbit-scan. Default directory is ~/.blindbit-scan.")
	flag.Parse()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	config.SetupConfigs(config.DirectoryPath)
	d, err := daemon.SetupDaemon(config.DirectoryPath)
	if err != nil {
		panic(err)
	}

	go func() {
		go d.ContinuousScan()
		err := server.StartNewServer(d)
		if err != nil {
			panic(err)
		}
	}()

	// wait for program stop signal
	<-interrupt
}
