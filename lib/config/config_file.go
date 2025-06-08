package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
	"github.com/taukakao/browser-glue/lib/util"
)

type NativeConfigFile struct {
	Path    string
	Content NativeMessagingConfig
	browser util.Browser
}

func (config *NativeConfigFile) Name() string {
	return filepath.Base(config.Path)
}

func (config *NativeConfigFile) GetBrowser() util.Browser {
	return config.browser
}

func (config *NativeConfigFile) Matches(other *NativeConfigFile) bool {
	return config.browser == other.browser && config.Path == other.Path
}

func (config *NativeConfigFile) IsEnabled() bool {
	enabledConfigs := settings.EnabledNativeConfigFiles(config.browser)
	enabled := slices.Contains(enabledConfigs, config.Name())

	if enabled && !config.flatpakFileExists() {
		logs.Info("writing flatpak config file", config.Name())
		err := config.writeConfigToFlatpakDir()
		if err != nil {
			err = fmt.Errorf("could not write config %s to flatpak directory: %w", config.Path, err)
			logs.Error(err)
		}
	}

	return enabled
}

func (config *NativeConfigFile) Enable() error {
	err := config.writeConfigToFlatpakDir()
	if err != nil {
		err = fmt.Errorf("could not write config to flatpak directory: %w", err)
		logs.Error(err)
		return err
	}
	err = settings.SetNativeConfigFileEnabled(config.browser, config.Name(), true)
	if err != nil {
		err = fmt.Errorf("could not change setting of config file: %w", err)
		logs.Error(err)
		return err
	}
	return nil
}

func (config *NativeConfigFile) Disable() error {
	err := config.deleteConfigInFlatpakDir()
	if err != nil {
		logs.Warn("config not deleted from flatpak dir", err)
	}
	err = settings.SetNativeConfigFileEnabled(config.browser, config.Name(), false)
	if err != nil {
		err = fmt.Errorf("could not change setting of config file: %w", err)
		logs.Error(err)
		return err
	}

	return nil
}

func (config *NativeConfigFile) flatpakConfigPath() string {
	filename := filepath.Base(config.Path)

	switch config.browser {
	case util.Firefox:
		return filepath.Join(util.GetHomeDirPath(), ".var", "app", config.browser.GetFlatpakId(), ".mozilla", "native-messaging-hosts", filename)
	case util.Floorp:
		return filepath.Join(util.GetHomeDirPath(), ".var", "app", config.browser.GetFlatpakId(), ".mozilla", "native-messaging-hosts", filename)
	case util.Chromium:
		return filepath.Join(util.GetHomeDirPath(), ".var", "app", config.browser.GetFlatpakId(), "config", "chromium", "NativeMessagingHosts", filename)
	case util.Brave:
		return filepath.Join(util.GetHomeDirPath(), ".var", "app", config.browser.GetFlatpakId(), "config", "BraveSoftware", "Brave-Browser", "NativeMessagingHosts", filename)
	default:
		panic("unsupported browser")
	}
}

func (config *NativeConfigFile) flatpakFileExists() bool {
	flatpakPath := config.flatpakConfigPath()
	_, err := os.Lstat(flatpakPath)
	return err == nil
}

func (config *NativeConfigFile) writeConfigToFlatpakDir() error {
	flatpakPath := config.flatpakConfigPath()
	if config.flatpakFileExists() {
		err := os.Remove(flatpakPath)
		if err != nil {
			logs.Warn("could not remove old config file:", err)
		}
	}

	flatpakConfig := config.Content.CreateCopy()
	flatpakConfig.ConvertToCustomConfig(config.browser)
	err := flatpakConfig.WriteFile(flatpakPath)
	if err != nil {
		err = fmt.Errorf("could not create native config in flatpak folder: %w", err)
		logs.Error(err)
		return err
	}

	return nil
}

func (config *NativeConfigFile) deleteConfigInFlatpakDir() error {
	flatpakPath := config.flatpakConfigPath()
	if !config.flatpakFileExists() {
		logs.Info("native config file", flatpakPath, "already deleted")
		return nil
	}
	err := os.Remove(flatpakPath)
	if err != nil {
		err = fmt.Errorf("could not remove config file: %w", err)
		logs.Error(err)
		return err
	}
	return nil
}

func CollectEnabledConfigFiles(browser util.Browser) ([]NativeConfigFile, error) {
	configFiles, err := CollectConfigFiles(browser)
	if err != nil {
		return configFiles, err
	}

	filteredConfigFiles := []NativeConfigFile{}

	for _, configFile := range configFiles {
		if configFile.IsEnabled() {
			filteredConfigFiles = append(filteredConfigFiles, configFile)
		}
	}

	return filteredConfigFiles, nil
}

func CollectConfigFiles(browser util.Browser) (configFiles []NativeConfigFile, err error) {
	if browser == util.AllBrowsers {
		for _, browser := range util.GetAllBrowsers() {
			newFiles, err := CollectConfigFiles(browser)
			if err != nil {
				return configFiles, err
			}
			configFiles = append(configFiles, newFiles...)
		}
		return
	}

	homePath := util.GetHomeDirPath()

	var hostFolderPath string
	switch browser {
	case util.Firefox, util.Floorp:
		hostFolderPath = filepath.Join(homePath, ".mozilla", "native-messaging-hosts")
	case util.Chromium, util.Brave:
		hostFolderPath = filepath.Join(homePath, ".config", "chromium", "NativeMessagingHosts")
	default:
		err = util.ErrBrowserNotKnown
		return
	}

	if _, err := os.Lstat(hostFolderPath); errors.Is(err, os.ErrNotExist) {
		logs.Warn("Host folder created. Logging out and back in might be requred.")
		os.MkdirAll(hostFolderPath, 0o755)
	}

	hostConfigFiles, err := collectConfigFilePathsInFolder(hostFolderPath)
	if err != nil {
		err = fmt.Errorf("can't find native messaging config files: %w", err)
		logs.Error(err)
		return
	}

	for _, hostConfigFile := range hostConfigFiles {
		decoded := NativeMessagingConfig{}
		err = decoded.ParseFile(hostConfigFile)
		if err != nil {
			err = fmt.Errorf("failed to parse config file %s: %w", hostConfigFile, err)
			logs.Error(err)
			continue
		}
		configFiles = append(configFiles, NativeConfigFile{Path: hostConfigFile, Content: decoded, browser: browser})
	}

	return configFiles, nil
}

func collectConfigFilePathsInFolder(folderPath string) ([]string, error) {
	var configFiles []string
	files, err := os.ReadDir(folderPath)
	if errors.Is(err, os.ErrNotExist) {
		return configFiles, nil
	}
	if err != nil {
		err = fmt.Errorf("Error reading directory %s: %w", folderPath, err)
		logs.Error(err)
		return configFiles, err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			path := filepath.Join(folderPath, file.Name())
			configFiles = append(configFiles, path)
		}
	}

	return configFiles, nil
}
