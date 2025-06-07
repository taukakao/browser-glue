package userapp_settings

import (
	_ "embed"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/taukakao/browser-glue/lib/config"
)

//go:embed userapp_settings.ui
var uiXML string

func NewUserappSettings(configFile config.NativeConfigFile) gtk.Widgetter {
	browser := configFile.GetBrowser()

	builder := gtk.NewBuilderFromString(uiXML)

	page := builder.GetObject("userapp_settings").Cast().(*adw.StatusPage)
	enableSwitch := builder.GetObject("enable_switch").Cast().(*adw.SwitchRow)
	execInfo := builder.GetObject("exec_info").Cast().(*adw.ActionRow)
	configPathInfo := builder.GetObject("config_path_info").Cast().(*adw.ActionRow)
	extensionsInfo := builder.GetObject("extensions_info").Cast().(*adw.ActionRow)
	browserInfo := builder.GetObject("browser_info").Cast().(*adw.ActionRow)

	page.SetTitle(configFile.Content.Name)
	page.SetDescription(configFile.Content.Description)

	enableSwitch.SetActive(configFile.IsEnabled())
	enableSwitch.Connect("notify::active", func(enableSwitch *adw.SwitchRow) {
		enable := enableSwitch.Active()

		if enable {
			configFile.Enable()
		} else {
			configFile.Disable()
		}
	})

	execInfo.SetSubtitle(configFile.Content.Executable)

	configPathInfo.SetSubtitle(configFile.Path)

	extensionsInfo.SetSubtitle(strings.Join(configFile.Content.GetExtensions(), "\n"))

	browserInfo.SetSubtitle(browser.GetName())

	return page
}
