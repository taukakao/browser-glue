package commands

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/server"
	"github.com/taukakao/browser-glue/lib/settings"
	"github.com/taukakao/browser-glue/lib/util"
)

var clientExecutableData []byte

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
	var err error = writeClientExecutable()
	if err != nil {
		pterm.Error.Println("Could not write client executable:", err)
		return 1
	}

	allServersExited := make(chan struct{})
	server.RunEnabledServersBackground(settings.AllBrowsers, *listenIn, allServersExited)

	// if errors.Is(err, server.ErrNoConfigFiles) {
	// 	pterm.Error.Println("You have not enabled any configs yet.")
	// 	return 1
	// } else if err != nil {
	// 	pterm.Error.Println("Could not start enabled servers:", err)
	// 	return 2
	// }

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

func writeClientExecutable() error {
	var err error

	clientExecutablePath := util.GetClientExecutablePath()

	err = os.MkdirAll(filepath.Dir(clientExecutablePath), 0o755)
	if err != nil {
		err = fmt.Errorf("can't create directory for client executable: %w", err)
		logs.Error(err)
		return err
	}

	file, err := os.Create(clientExecutablePath)
	if err != nil {
		err = fmt.Errorf("can't create client executable file: %w", err)
		logs.Error(err)
		return err
	}
	defer file.Close()

	err = file.Chmod(0o755)
	if err != nil {
		err = fmt.Errorf("can't change permissions for client executable file: %w", err)
		logs.Error(err)
		return err
	}

	_, err = file.Write(clientExecutableData)
	if err != nil {
		err = fmt.Errorf("can't write client executable: %w", err)
		logs.Error(err)
		return err
	}

	logs.Info("client executable created in:", clientExecutablePath)

	return nil
}

var listenIn *bool

func init() {
	listenIn = serverCmd.PersistentFlags().BoolP("listen-in", "l", false, "print out messages that are sent through this program")
}
