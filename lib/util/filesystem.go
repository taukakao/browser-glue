package util

// don't use any packages from this repo or otherwise not in the stdlib
// this gets included in the client so it needs to be as small as possible
import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

var ErrBrowserNotKnown = errors.New("this browser is not known")

type Browser string

const (
	NoneBrowser Browser = ""
	AllBrowsers Browser = "all"
	Firefox     Browser = "firefox"
)

func (browser *Browser) GetFlatpakId() string {
	switch *browser {
	case Firefox:
		return "org.mozilla.firefox"
	default:
		panic(ErrBrowserNotKnown)
	}
}

func GetAllBrowsers() []Browser {
	return []Browser{Firefox}
}

func GenerateSocketPath(browserId string, extensionName string) string {
	socketNameEncoded := socketEncoding.EncodeToString([]byte(extensionName))
	socketFileName := fmt.Sprintf("%s.socket", socketNameEncoded)

	newSocketDir := filepath.Join(runtimeDir, "app", browserId, shortAppId)

	return filepath.Join(newSocketDir, socketFileName)
}

func GetHomeDirPath() string {
	return homeDir
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

func MakePathHomeRelative(path string) string {
	pathRel, err := filepath.Rel(homeDir, clientExecutableDir)
	if err != nil {
		return path
	}

	return filepath.Join("~", pathRel)
}

func findHomeDirPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		currentUser, err := user.Current()
		if err != nil || currentUser.Username == "" {
			// If this happens then something is very wrong with the system
			err = fmt.Errorf("could not find the home dir: %w", err)
			panic(err)
		}
		homeDir = filepath.Join("/", "home", currentUser.Username)
	}
	return homeDir
}

func findUserDataDirPath() string {
	dataDir, ok := os.LookupEnv("XDG_DATA_HOME")
	if !ok {
		homeDir := homeDir
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	return dataDir
}

func findUserConfigDir() string {
	configDir, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok {
		homeDir := homeDir
		configDir = filepath.Join(homeDir, ".config")
	}
	return configDir
}

func findRuntimeDir() string {
	runtimeDir, ok := os.LookupEnv("XDG_RUNTIME_DIR")
	if !ok {
		user, err := user.Current()
		if err != nil {
			err = fmt.Errorf("could not find the runtime dir: %w", err)
			panic(err)
		}
		runtimeDir = filepath.Join("/", "run", "user", user.Uid)
	}
	return runtimeDir
}

var shortAppId = "browser-glue"
var homeDir string = findHomeDirPath()
var runtimeDir string = findRuntimeDir()
var customUserDataDir string = filepath.Join(findUserDataDirPath(), shortAppId)
var customUserConfigDir string = filepath.Join(findUserConfigDir(), shortAppId)
var clientExecutableDir string = filepath.Join(customUserDataDir, "client")
var clientExecutablePath string = filepath.Join(clientExecutableDir, "client")
var socketEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-")
