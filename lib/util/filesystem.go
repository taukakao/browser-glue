package util

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/taukakao/browser-glue/lib/logs"
)

var CustomDataDir string
var ClientExecutablePath string

func GenerateSocketPath(extensionName string) string {
	socketNameEncoded := socketEncoding.EncodeToString([]byte(extensionName))
	return fmt.Sprintf(socketPathFormat, socketNameEncoded)
}

func FindHomeDirPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
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

func FindUserDataDirPath() string {
	dataDir, ok := os.LookupEnv("XDG_DATA_HOME")
	if !ok {
		homeDir := FindHomeDirPath()
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	return dataDir
}

var socketPathFormat string
var socketEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-")

func init() {
	dataDir := FindUserDataDirPath()

	CustomDataDir = filepath.Join(dataDir, "browser-glue")
	socketPathFormat = filepath.Join(CustomDataDir, "%s.socket")

	ClientExecutablePath = filepath.Join(CustomDataDir, "client")
}
