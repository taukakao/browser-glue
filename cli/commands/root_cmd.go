package commands

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "Command to connect browser extensions with applications.",
	Long:  `Browser Glue is an application that allows users to connect their browser extensions to locally running applications.`,
}

func askForBrowser() (util.Browser, int) {
	options := []string{}
	allBrowsers := util.GetAllBrowsers()
	for _, browser := range allBrowsers {
		options = append(options, string(browser))
	}
	selected, err := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		Show()

	if err != nil {
		logs.Error(err)
		return "", 1
	}

	selectedIndex := slices.Index(options, selected)
	if selectedIndex == -1 {
		return "", 1
	}

	return allBrowsers[selectedIndex], 0
}

type BrowserValue struct {
	Browser util.Browser
}

func (bv *BrowserValue) String() string {
	switch bv.Browser {
	case util.Firefox:
		return "firefox"
	default:
		return ""
	}
}

func (bv *BrowserValue) Set(input string) error {
	input = strings.ToLower(input)
	switch input {
	case "firefox":
		bv.Browser = util.Firefox
	default:
		return errors.New("unsupported browser, supported browsers are: firefox")
	}
	return nil
}

func (bv *BrowserValue) Type() string {
	return "string"
}

var selectedBrowserFlag BrowserValue

func init() {
	rootCmd.PersistentFlags().VarP(&selectedBrowserFlag, "browser", "b", "select browser")

	rootCmd.AddCommand(appsCmd)
	rootCmd.AddCommand(serverCmd)
}
