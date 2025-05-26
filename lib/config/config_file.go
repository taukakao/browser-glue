package config

import (
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
	browser settings.Browser
}

func (config *NativeConfigFile) Name() string {
	return filepath.Base(config.Path)
}

func (config *NativeConfigFile) IsEnabled() bool {
	enabledConfigs := settings.EnabledNativeConfigFiles(config.browser)
	enabled := slices.Contains(enabledConfigs, config.Name())
	if enabled && !config.flatpakFileExists() {
		err := config.writeConfigToFlatpakDir()
		if err != nil {
			logs.Error("could not write config", config.Path, "to flatpak directory:", err)
		}
	}
	return enabled

}

func (config *NativeConfigFile) Enable() error {
	err := settings.SetNativeConfigFileEnabled(config.browser, config.Path, true)
	if err != nil {
		logs.Error("could not change setting of config file:", err)
		return err
	}
	err = config.writeConfigToFlatpakDir()
	if err != nil {
		logs.Error("could not write config to flatpak directory:", err)
		return err
	}
	return nil
}

func (config *NativeConfigFile) Disable() error {
	err := settings.SetNativeConfigFileEnabled(config.browser, config.Path, false)
	if err != nil {
		logs.Error("could not change setting of config file:", err)
		return err
	}
	err = config.deleteConfigInFlatpakDir()
	if err != nil {
		logs.Warn("config not deleted from flatpak dir", err)
	}
	return nil
}

func (config *NativeConfigFile) flatpakConfigPath() string {
	if config.browser != settings.Firefox {
		panic("unsupported browser")
	}
	filename := filepath.Base(config.Path)
	return filepath.Join(util.FindHomeDirPath(), ".var", "app", "org.mozilla.firefox", ".mozilla", "native-messaging-hosts", filename)
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

	err := config.Content.WriteFile(flatpakPath)
	if err != nil {
		logs.Error("could not create native config in flatpak folder", err)
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
		logs.Error("could not remove config file:", err)
		return err
	}
	return nil
}

func CollectEnabledConfigFiles(browser settings.Browser) ([]NativeConfigFile, error) {
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

func CollectConfigFiles(browser settings.Browser) (configFiles []NativeConfigFile, err error) {
	homePath := util.FindHomeDirPath()

	var hostFolderPath string
	switch browser {
	case settings.Firefox:
		hostFolderPath = filepath.Join(homePath, ".mozilla", "native-messaging-hosts")
	default:
		err = fmt.Errorf("unsupported Browser")
		return
	}

	hostConfigFiles, err := collectConfigFilePathsInFolder(hostFolderPath)
	if err != nil {
		logs.Error("can't find native messaging config files", err)
		return
	}

	for _, hostConfigFile := range hostConfigFiles {
		decoded := NativeMessagingConfig{}
		err = decoded.ParseFile(hostConfigFile)
		if err != nil {
			logs.Error("failed to parse config file", hostConfigFile, ":", err)
			return
		}
		configFiles = append(configFiles, NativeConfigFile{Path: hostConfigFile, Content: decoded, browser: browser})
	}

	return configFiles, nil
}

func collectConfigFilePathsInFolder(folderPath string) ([]string, error) {
	var configFiles []string
	files, err := os.ReadDir(folderPath)
	if err != nil {
		logs.Error("Error reading directory:", err)
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
