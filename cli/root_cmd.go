package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
)

var rootCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "Command to connect browser extensions with applications.",
	Long:  `Browser Glue is an application that allows users to connect their browser extensions to locally running applications.`,
}

func askForBrowser() (settings.Browser, int) {
	options := []string{"firefox"}
	selected, err := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		Show()

	if err != nil {
		logs.Error(err)
		return "", 1
	}

	switch selected {
	case "firefox":
		return settings.Firefox, 0
	default:
		return "", 1
	}
}

type BrowserValue struct {
	Browser settings.Browser
}

func (bv *BrowserValue) String() string {
	switch bv.Browser {
	case settings.Firefox:
		return "firefox"
	default:
		return ""
	}
}

func (bv *BrowserValue) Set(input string) error {
	input = strings.ToLower(input)
	switch input {
	case "firefox":
		bv.Browser = settings.Firefox
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
	extensionsCmd.PersistentFlags().VarP(&selectedBrowserFlag, "browser", "b", "select browser")

	rootCmd.AddCommand(extensionsCmd)
	rootCmd.AddCommand(serverCmd)
}
