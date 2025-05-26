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

// TODO: use this
// func startServer() int {
// 	customMultiselect := pterm.
// 		DefaultInteractiveMultiselect.
// 		WithFilter(false).
// 		WithKeyConfirm(keys.Enter).
// 		WithKeySelect(keys.Space).
// 		WithDefaultText("Select which servers to enable")

// 	configFiles, err := config.CollectConfigFiles()
// 	if err != nil {
// 		logs.Error("could not find config files", err)
// 		return 1
// 	}
// 	if len(configFiles) == 0 {
// 		logs.Error("could not find any native messaging configs")
// 		return 1
// 	}

// 	configFileNames := make([]string, 0, len(configFiles))
// 	enabledConfigFileNames := make([]string, 0, len(configFileNames))
// 	for _, config := range configFiles {
// 		configName := config.Name()
// 		configFileNames = append(configFileNames, configName)
// 		if config.IsEnabled() {
// 			enabledConfigFileNames = append(enabledConfigFileNames, configName)
// 		}
// 	}

// 	selectedConfigs, err := customMultiselect.
// 		WithOptions(configFileNames).
// 		WithDefaultOptions(enabledConfigFileNames).
// 		Show()

// 	if err != nil {
// 		logs.Error(err)
// 		return 1
// 	}

// 	if len(selectedConfigs) == 0 {
// 		pterm.Error.Println("No arguments given")
// 		return 1
// 	}

// 	c := make(chan os.Signal, 1)
// 	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

// 	for _, config := range configFiles {
// 		configFileName := config.Name()
// 		if !slices.Contains(selectedConfigs, configFileName) {
// 			continue
// 		}
// 		for _, extension := range config.Content.AllowedExtensions {
// 			pterm.Info.Println("Running server for", config.Content.Name)
// 			exitChan := make(chan error)
// 			server.RunServerBackground(config, extension, exitChan)
// 			go func() {
// 				err := <-exitChan
// 				if err != nil {
// 					pterm.Error.Println("failed to run server for", config.Content.Name, ":", err)
// 				}
// 			}()
// 		}
// 	}

// 	<-c
// 	pterm.Info.Println("cleaning up, press Ctrl+C again to force close")

// 	go func() {
// 		<-c
// 		os.Exit(1)
// 	}()

// 	server.StopServers()
// 	return 0
// }
