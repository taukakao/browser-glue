package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

func cacheBrowserIcons() {
	var iconCacheDir = filepath.Join(util.GetCustomUserCacheDir(), "browsericons")
	err := os.MkdirAll(iconCacheDir, 0o775)
	if err != nil {
		logs.Warn("could not create cache file", iconCacheDir, ":", err)
	}

	for _, browser := range util.GetAllBrowsers() {
		exists, err := checkIfIconExists(iconCacheDir, browser.GetFlatpakId())
		if err == nil && exists {
			continue
		}
		iconUrl, err := fetchIconUrl(browser)
		if err != nil {
			logs.Warn("could not download browser icon:", err)
			continue
		}
		err = downloadFile(iconCacheDir, iconUrl)
		if err != nil {
			logs.Warn("could not download browser icon:", err)
		}
	}
	<-activated
	glib.IdleAdd(func() {
		gtk.IconThemeGetForDisplay(gdk.DisplayGetDefault()).AddSearchPath(iconCacheDir)
	})

}

type flathubApiResponse struct {
	Icon string `json:"icon"`
}

func fetchIconUrl(browser util.Browser) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://flathub.org/api/v2/appstream/%s", browser.GetFlatpakId()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result flathubApiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Icon == "" {
		return "", errors.New("no icon url received")
	}

	return result.Icon, nil
}

func downloadFile(directory string, fileUrl string) error {
	parsedURL, err := url.Parse(fileUrl)
	if err != nil {
		return err
	}
	fileName := path.Base(parsedURL.Path)
	writeFilePath := filepath.Join(directory, fileName)

	resp, err := http.Get(fileUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(writeFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func checkIfIconExists(directory string, iconName string) (bool, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if iconName+filepath.Ext(entry.Name()) == entry.Name() {
			return true, nil
		}
	}
	return false, nil
}
