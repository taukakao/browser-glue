package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/server"
	"github.com/taukakao/browser-glue/lib/settings"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the server",
	Long:  `Run the server`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := startServer()
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

func startServer() int {
	exitChan := make(chan error)
	err := server.RunEnabledServersBackground(settings.Firefox, exitChan)

	if errors.Is(err, server.ErrNoConfigFiles) {
		pterm.Error.Println("You have not enabled any configs yet.")
		return 1
	} else if err != nil {
		pterm.Error.Println("Could not start enabled servers:", err)
		return 2
	}

	pterm.Info.Println("Servers started")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt
	pterm.Info.Println("cleaning up, press Ctrl+C again to force close")

	go func() {
		<-interrupt
		os.Exit(3)
	}()

	server.StopServers()
	return 0
}
