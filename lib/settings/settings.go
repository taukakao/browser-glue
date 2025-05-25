package settings

import (
	"errors"
	"os"
	"slices"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

type Browser string

const (
	Firefox Browser = "firefox"
)

func SubscribeToChanges(subscription chan struct{}) {
	if subscription == nil {
		return
	}
	subscribers = append(subscribers, subscription)
}

func EnabledNativeConfigFiles(browser Browser) []string {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	return viper.GetStringSlice(string(browser) + ".enabledConfigs")
}

func SetNativeConfigFileEnabled(browser Browser, nativeConfigFilePath string, enable bool) {
	viperMutex.Lock()
	defer viperMutex.Unlock()

	allEnabled := viper.GetStringSlice(string(browser) + ".enabledConfigs")
	isCurrentlyEnabled := slices.Contains(allEnabled, nativeConfigFilePath)
	if isCurrentlyEnabled == enable {
		return
	}

	if enable {
		allEnabled = append(allEnabled, nativeConfigFilePath)
	} else {
		allEnabled = slices.DeleteFunc(allEnabled, func(element string) bool { return element == nativeConfigFilePath })
	}

	viper.Set(string(browser)+".enabledConfigs", allEnabled)
	viper.WriteConfig()
}

var viperMutex sync.Mutex

var subscribers []chan struct{}

func init() {
	var err error

	viperMutex.Lock()
	defer viperMutex.Unlock()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	userConfigDir := util.GetCustomUserConfigDir()
	viper.AddConfigPath(userConfigDir)
	err = viper.ReadInConfig()
	if errors.As(err, &viper.ConfigFileNotFoundError{}) {
		logs.Info("creating config file", viper.ConfigFileUsed())
		err = os.MkdirAll(userConfigDir, 0o775)
		if err != nil {
			logs.Error(err)
		}
		err = viper.SafeWriteConfig()
		if err != nil {
			logs.Error("could not write config file", err)
			panic(err)
		}
	} else if err != nil {
		logs.Error("could not read config", viper.ConfigFileUsed(), err)
		panic(err)
	}

	viper.OnConfigChange(onSettingsFileChanged)
	viper.WatchConfig()
}

func onSettingsFileChanged(e fsnotify.Event) {
	for _, subscriber := range subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
			// ignore this subscriber if not listening
		}
	}
	logs.Info("config changed, reloading config")
}
