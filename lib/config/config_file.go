package config

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

type NativeConfigFile struct {
	Path      string
	isEnabled bool
	Content   NativeMessagingConfig
}

func (config *NativeConfigFile) Name() string {
	return filepath.Base(config.Path)
}

func (config *NativeConfigFile) IsEnabled() bool {
	return config.isEnabled
}

func CollectConfigFiles() (configFiles []NativeConfigFile, err error) {
	homePath := util.FindHomeDirPath()
	hostFolderPath := filepath.Join(homePath, ".mozilla", "native-messaging-hosts")
	flatpakFolderPath := filepath.Join(homePath, ".var", "app", "org.mozilla.firefox", ".mozilla", "native-messaging-hosts")

	hostConfigFiles, err := collectConfigFilePathsInFolder(hostFolderPath)
	if err != nil {
		logs.Error("can't find native messaging config files", err)
		return
	}
	flatpakConfigFiles, err := collectConfigFilePathsInFolder(flatpakFolderPath)
	if err != nil {
		logs.Error("can't asses which config files are enabled", err)
		return
	}

	for _, hostConfigFile := range hostConfigFiles {
		isEnabled := slices.Contains(flatpakConfigFiles, hostConfigFile)
		decoded := NativeMessagingConfig{}
		err = decoded.ParseFile(hostConfigFile)
		if err != nil {
			logs.Error("failed to parse config file", hostConfigFile, ":", err)
			return
		}
		configFiles = append(configFiles, NativeConfigFile{Path: hostConfigFile, isEnabled: isEnabled, Content: decoded})
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
