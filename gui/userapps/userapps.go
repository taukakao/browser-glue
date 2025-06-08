package userapps

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/taukakao/browser-glue/gui/resources"
	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/util"
)

func NewUserappsList(browser util.Browser, clickItemCallback func(config.NativeConfigFile)) gtk.Widgetter {
	builder := resources.GetBuilderForPath("userapps/userapps.ui")

	userappsList := builder.GetObject("userapps").Cast().(*adw.Clamp)
	userappsPage := builder.GetObject("userapps_page").Cast().(*adw.StatusPage)
	configList := builder.GetObject("config_list").Cast().(*adw.PreferencesGroup)

	userappsPage.SetIconName(browser.GetFlatpakId())

	userappsPage.SetTitle(fmt.Sprintf("Native Applications for %s", browser.GetName()))

	configFiles, err := config.CollectConfigFiles(browser)
	if err != nil {
		panic(err)
	}
	for _, configFile := range configFiles {
		rowBuilder := resources.GetBuilderForPath("userapps/widget_row.ui")

		actionRow := rowBuilder.GetObject("row").Cast().(*adw.ActionRow)
		actionRow.SetTitle(configFile.Content.Name)
		actionRow.SetSubtitle(configFile.Content.Description)

		actionRow.Connect("activated", func() {
			clickItemCallback(configFile)
		})

		configList.Add(actionRow)
	}

	return userappsList
}
