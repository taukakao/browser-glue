package util

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/taukakao/browser-glue/lib/logs"
)

func GenerateSocketPath(socketDir string, extensionName string) string {
	socketNameEncoded := socketEncoding.EncodeToString([]byte(extensionName))
	socketFileName := fmt.Sprintf("%s.socket", socketNameEncoded)
	return filepath.Join(socketDir, socketFileName)
}

func FindHomeDirPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		currentUser, err := user.Current()
		if err != nil || currentUser.Username == "" {
			// If this happens then something is very wrong with the system
			logs.Error("could not find the home dir:", err)
			panic(fmt.Sprintln("can't continue without knowing the home dir"))
		}
		homeDir = filepath.Join("/", "home", currentUser.Username)
	}
	return homeDir
}

func GetCustomUserDataDir() string {
	return customUserDataDir
}

func GetCustomUserConfigDir() string {
	return customUserConfigDir
}

func GetClientExecutableDir() string {
	return clientExecutableDir
}

func GetClientExecutablePath() string {
	return clientExecutablePath
}

func findUserDataDirPath() string {
	dataDir, ok := os.LookupEnv("XDG_DATA_HOME")
	if !ok {
		homeDir := FindHomeDirPath()
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	return dataDir
}

func findUserConfigDir() string {
	configDir, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok {
		homeDir := FindHomeDirPath()
		configDir = filepath.Join(homeDir, ".config")
	}
	return configDir
}

var shortAppId = "browser-glue"
var customUserDataDir string = filepath.Join(findUserDataDirPath(), shortAppId)
var customUserConfigDir string = filepath.Join(findUserConfigDir(), shortAppId)
var clientExecutableDir string = filepath.Join(customUserDataDir, "client")
var clientExecutablePath string = filepath.Join(clientExecutableDir, "client")
var socketEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-")
