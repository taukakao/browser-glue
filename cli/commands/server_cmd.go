package commands

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/server"
	"github.com/taukakao/browser-glue/lib/util"
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
	browser := util.AllBrowsers
	if selectedBrowserFlag.Browser != util.NoneBrowser {
		browser = selectedBrowserFlag.Browser
	}

	allServersExited := make(chan struct{})
	server.RunEnabledServersBackground(browser, *listenIn, allServersExited)

	pterm.Info.Println("Servers started")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-allServersExited:
		pterm.Error.Println("All servers exited!")
		return 5
	case <-interrupt:
	}
	pterm.Info.Println("cleaning up, press Ctrl+C again to force close")

	go func() {
		<-interrupt
		os.Exit(3)
	}()

	server.StopServers()
	return 0
}

var listenIn *bool

func init() {
	listenIn = serverCmd.PersistentFlags().BoolP("listen-in", "l", false, "print out messages that are sent through this program")
}
