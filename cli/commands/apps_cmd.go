package commands

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"atomicgo.dev/keyboard/keys"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Configure apps",
	Long:  `Select if apps should be enabled or disabled.`,
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List apps",
	Long:  `Print out a list of all apps and information about them.`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := listApps(selectedBrowserFlag.Browser)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

var appsSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select apps",
	Long:  `Choose from a list of available apps.`,
	Run: func(cmd *cobra.Command, args []string) {
		exitCode := selectApps(selectedBrowserFlag.Browser)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	},
}

func listApps(browser settings.Browser) int {
	if browser == settings.NoneBrowser {
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

	data := [][]string{{"App Config Name", "Enabled", "Supported Extensions"}}

	for _, configFile := range configFiles {
		newLine := []string{configFile.Name(), fmt.Sprint(configFile.IsEnabled()), strings.Join(configFile.Content.AllowedExtensions, " | ")}
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

func selectApps(browser settings.Browser) int {
	if browser == settings.NoneBrowser {
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
		WithDefaultText("Select which apps to enable").
		WithOptions(configFileNames).
		WithDefaultOptions(enabledConfigFileNames).
		Show()

	if err != nil {
		logs.Error(err)
		return 1
	}

	finalErrCode := 0

	enableConfigs := []config.NativeConfigFile{}
	disableConfigs := []config.NativeConfigFile{}

	for _, config := range configFiles {
		enable := slices.Contains(selectedConfigs, config.Name())
		if enable == config.IsEnabled() {
			continue
		}

		if enable {
			enableConfigs = append(enableConfigs, config)
		} else {
			disableConfigs = append(disableConfigs, config)
		}

	}

	for _, config := range enableConfigs {
		err = config.Enable()

		if err != nil {
			pterm.Error.Println("Failed to enable app", config.Name(), ":", err)
			finalErrCode = 1
			continue
		}

		pterm.Info.Println("App", config.Name(), "enabled.")
	}
	for _, config := range disableConfigs {
		err = config.Disable()

		if err != nil {
			pterm.Error.Println("Failed to disable app", config.Name(), ":", err)
			finalErrCode = 1
			continue
		}

		pterm.Info.Println("App", config.Name(), "disabled.")
	}

	pterm.Info.Println("Server will be reloaded automatically if it's running.")

	return finalErrCode
}

func collectConfigFiles(browser settings.Browser) ([]config.NativeConfigFile, []string, []string, int) {
	configFiles, err := config.CollectConfigFiles(browser)
	if err != nil {
		err = fmt.Errorf("problem while looking for app configuration files: %w", err)
		logs.Error(err)
		return configFiles, []string{}, []string{}, 1
	}
	if len(configFiles) == 0 {
		pterm.Error.Println("Could not find any app configuration files.")
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
	appsCmd.AddCommand(appsListCmd)
	appsCmd.AddCommand(appsSelectCmd)
}
