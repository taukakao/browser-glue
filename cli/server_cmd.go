package main

import (
	"os"
	"os/signal"
	"slices"
	"syscall"

	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/server"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the server",
	Long:  `Run the server`,
	Run:   startServer,
}

func startServer(cmd *cobra.Command, args []string) {
	exitCode := 0
	defer func() {
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	customMultiselect := pterm.
		DefaultInteractiveMultiselect.
		WithFilter(false).
		WithKeyConfirm(keys.Enter).
		WithKeySelect(keys.Space).
		WithDefaultText("Select which servers to enable")

	configFiles, err := config.CollectConfigFiles()
	if err != nil {
		logs.Error("could not find config files", err)
		exitCode = 1
		return
	}
	if len(configFiles) == 0 {
		logs.Error("could not find any native messaging configs")
		exitCode = 1
		return
	}

	configFileNames := make([]string, 0, len(configFiles))
	enabledConfigFileNames := make([]string, 0, len(configFileNames))
	for _, config := range configFiles {
		configName := config.Name()
		configFileNames = append(configFileNames, configName)
		if config.IsEnabled() {
			enabledConfigFileNames = append(enabledConfigFileNames, configName)
		}
	}

	selectedConfigs, err := customMultiselect.
		WithOptions(configFileNames).
		WithDefaultOptions(enabledConfigFileNames).
		Show()

	if err != nil {
		logs.Error(err)
		exitCode = 1
		return
	}

	if len(selectedConfigs) == 0 {
		pterm.Error.Println("No arguments given")
		exitCode = 1
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	for _, config := range configFiles {
		configFileName := config.Name()
		if !slices.Contains(selectedConfigs, configFileName) {
			continue
		}
		for _, extension := range config.Content.AllowedExtensions {
			pterm.Info.Println("Running server for", config.Content.Name)
			exitChan := make(chan error)
			server.RunServerBackground(config, extension, exitChan)
			go func() {
				err := <-exitChan
				if err != nil {
					pterm.Error.Println("failed to run server for", config.Content.Name, ":", err)
				}
			}()
		}
	}

	<-c
	pterm.Info.Println("cleaning up, press Ctrl+C again to force close")

	go func() {
		<-c
		os.Exit(1)
	}()

	server.StopServers()
	return
}
