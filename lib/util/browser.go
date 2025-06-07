package util

import (
	"errors"
	"path/filepath"
)

var ErrBrowserNotKnown = errors.New("this browser is not known")

type Browser string

const (
	NoneBrowser Browser = ""
	AllBrowsers Browser = "all"
	Firefox     Browser = "firefox"
	Floorp      Browser = "floorp"
	Chromium    Browser = "chromium"
	Brave       Browser = "brave"
)

var allBrowsers = []Browser{Firefox, Floorp, Chromium, Brave}

func (browser *Browser) GetFlatpakId() string {
	switch *browser {
	case Firefox:
		return "org.mozilla.firefox"
	case Floorp:
		return "one.ablaze.floorp"
	case Chromium:
		return "org.chromium.Chromium"
	case Brave:
		return "com.brave.Browser"
	default:
		panic(ErrBrowserNotKnown)
	}
}

func (browser *Browser) GetName() string {
	switch *browser {
	case Firefox:
		return "Firefox"
	case Floorp:
		return "Floorp"
	case Chromium:
		return "Chromium"
	case Brave:
		return "Brave"
	default:
		panic(ErrBrowserNotKnown)
	}
}

func (browser *Browser) GetFlatpakRuntimeAppFolder() string {
	return filepath.Join(runtimeDir, "app", browser.GetFlatpakId(), shortAppId)
}

func (browser *Browser) GetClientPath() string {
	return filepath.Join(browser.GetFlatpakRuntimeAppFolder(), "client")
}

func GetAllBrowsers() []Browser {
	return allBrowsers
}
