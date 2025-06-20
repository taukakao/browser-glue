package util

// don't use any packages from this repo or otherwise not in the stdlib
// this gets included in the client so it needs to be as small as possible
import (
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func GenerateSocketFileName(extensionName string) string {
	extensionName = strings.TrimPrefix(extensionName, "chrome-extension://")
	socketNameEncoded := socketEncoding.EncodeToString([]byte(extensionName))

	if len(socketNameEncoded) >= 50 {
		socketNameEncoded = socketNameEncoded[:50]
	}

	return socketNameEncoded
}

func GetHomeDirPath() string {
	return homeDir
}

func GetCustomUserConfigDir() string {
	return customUserConfigDir
}

func GetCustomUserCacheDir() string {
	return customUserCacheDir
}

func MakePathHomeRelative(path string) string {
	pathRel, err := filepath.Rel(homeDir, path)
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
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	return dataDir
}

func findUserConfigDir() string {
	configDir, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok {
		configDir = filepath.Join(homeDir, ".config")
	}
	return configDir
}

func findUserCacheDir() string {
	cacheDir, ok := os.LookupEnv("XDG_CACHE_HOME")
	if !ok {
		cacheDir = filepath.Join(homeDir, ".cache")
	}
	return cacheDir
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

func GetLongAppId() string {
	return longAppId
}

var (
	shortAppId = "browser-glue"
	longAppId  = "net.taukakao.BrowserGlue"
)

var (
	homeDir             string = findHomeDirPath()
	runtimeDir          string = findRuntimeDir()
	customUserDataDir   string = filepath.Join(findUserDataDirPath(), shortAppId)
	customUserConfigDir string = filepath.Join(findUserConfigDir(), shortAppId)
	customUserCacheDir  string = filepath.Join(findUserCacheDir(), shortAppId)
)

var socketEncoding = base64.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-")
