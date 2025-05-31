package main

import (
	"fmt"
	"os"
	"slices"

	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
)

var extensionsCmd = &cobra.Command{
	Use:   "extensions",
	Short: "Configure extensions",
	Long:  `Select if extensions should be enabled or disabled.`,
}

var extensionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List extensions",
	Long:  `Print out a list of all extensions and information about them.`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := listExtensions(selectedBrowserFlag.Browser)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

var extensionsSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select extensions",
	Long:  `Choose from a list of available extensions.`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := selectExtensions(selectedBrowserFlag.Browser)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

func listExtensions(browser settings.Browser) int {
	if browser == "" {
		browserNew, exitCode := askForBrowser()
		if exitCode != 0 {
			return exitCode
		}
		browser = browserNew
	}

	configFiles, _, _, exitCode := collectConfigFiles(browser)
	if exitCode != 0 {
		return exitCode
	}

	data := [][]string{{"Extension Config Name", "Enabled", "Included Extensions"}}

	for _, configFile := range configFiles {
		newLine := []string{configFile.Name(), fmt.Sprint(configFile.IsEnabled()), fmt.Sprint(configFile.Content.AllowedExtensions)}
		data = append(data, newLine)
	}

	pterm.DefaultTable.
		WithHasHeader(true).
		WithRowSeparator("-").
		WithHeaderRowSeparator("-").
		WithData(data).
		Render()

	return 0
}

func selectExtensions(browser settings.Browser) int {
	if browser == "" {
		browserNew, exitCode := askForBrowser()
		if exitCode != 0 {
			return exitCode
		}
		browser = browserNew
	}

	var err error
	customMultiselect := pterm.
		DefaultInteractiveMultiselect.
		WithFilter(false).
		WithKeyConfirm(keys.Enter).
		WithKeySelect(keys.Space)

	configFiles, configFileNames, enabledConfigFileNames, exitCode := collectConfigFiles(browser)
	if exitCode != 0 {
		return exitCode
	}

	selectedConfigs, err := customMultiselect.
		WithDefaultText("Select which extensions to enable").
		WithOptions(configFileNames).
		WithDefaultOptions(enabledConfigFileNames).
		Show()

	if err != nil {
		logs.Error(err)
		return 1
	}

	finalErrCode := 0

	for _, config := range configFiles {
		enable := slices.Contains(selectedConfigs, config.Name())
		if enable == config.IsEnabled() {
			continue
		}

		if enable {
			err = config.Enable()

			if err != nil {
				pterm.Error.Println("Failed to enable extension", config.Name(), ":", err)
				finalErrCode = 1
				continue
			}

			pterm.Info.Println("Extension", config.Name(), "enabaled")
		} else {
			err = config.Disable()

			if err != nil {
				pterm.Error.Println("Failed to disable extension", config.Name(), ":", err)
				finalErrCode = 1
				continue
			}

			pterm.Info.Println("Extension", config.Name(), "disabled")
		}
	}

	pterm.Info.Println("Server will be reloaded automatically if it's running")

	return finalErrCode
}

func collectConfigFiles(browser settings.Browser) ([]config.NativeConfigFile, []string, []string, int) {
	configFiles, err := config.CollectConfigFiles(browser)
	if err != nil {
		err = fmt.Errorf("problem while looking for extension config files: %w", err)
		logs.Error(err)
		return configFiles, []string{}, []string{}, 1
	}
	if len(configFiles) == 0 {
		pterm.Error.Println("Could not find any extension config files")
		return configFiles, []string{}, []string{}, 1
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
	return configFiles, configFileNames, enabledConfigFileNames, 0
}

func init() {
	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsSelectCmd)
}
