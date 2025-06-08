package application

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/taukakao/browser-glue/gui/userapp_settings"
	"github.com/taukakao/browser-glue/gui/userapps"
	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/util"
)

func RunApplication(gresourceData []byte) {
	app := adw.NewApplication(util.GetLongAppId(), gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c

		glib.IdleAdd(func() {
			app.ActiveWindow().Close()
		})

		<-c
		panic("interrupted while cleaning up")
	}()

	go cacheBrowserIcons()

	gresource, err := gio.NewResourceFromData(glib.NewBytes(gresourceData))
	if err != nil {
		panic(fmt.Errorf("Could not load gresources: %w", err))
	}
	gio.ResourcesRegister(gresource)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

var activated = make(chan struct{}, 1)

func activate(app *adw.Application) {
	activated <- struct{}{}

	builder := gtk.NewBuilderFromResource("/net/taukakao/BrowserGlue/generated/application/main.ui")

	window := builder.GetObject("main_window").Cast().(*adw.ApplicationWindow)
	browserMenu := builder.GetObject("browser_select_menu").Cast().(*gio.Menu)
	navView := builder.GetObject("navigation_view").Cast().(*adw.NavigationView)
	userappsView := builder.GetObject("userapps_view").Cast().(*adw.ToolbarView)
	userappSettingsView := builder.GetObject("userapp_settings_toolbar").Cast().(*adw.ToolbarView)

	allBrowsers := util.GetAllBrowsers()
	for _, browser := range allBrowsers {
		browserMenu.Append(browser.GetName(), fmt.Sprintf("app.browser::%s", browser))
	}
	browserAction := gio.NewSimpleAction("browser", glib.NewVariantType("s"))
	app.AddAction(browserAction)
	browserAction.Connect("activate", func(action *gio.SimpleAction, target *glib.Variant) {
		browser := util.Browser(target.String())
		changeBrowser(userappsView, navView, userappSettingsView, browser)
	})

	changeBrowser(userappsView, navView, userappSettingsView, util.Firefox)

	app.AddWindow(&window.Window)
	window.SetVisible(true)
}

func changeBrowser(userappsView *adw.ToolbarView, navView *adw.NavigationView, userappSettingsView *adw.ToolbarView, browser util.Browser) {

	userappsList := userapps.NewUserappsList(browser, func(configFile config.NativeConfigFile) {
		displayUserapps(userappSettingsView, configFile)
		navView.PushByTag("native_app_settings")
	})

	userappsView.SetContent(userappsList)
}

func displayUserapps(navPage *adw.ToolbarView, configFile config.NativeConfigFile) {

	userappSettings := userapp_settings.NewUserappSettings(configFile)

	navPage.SetContent(userappSettings)
}
