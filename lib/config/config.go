package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

type ConfigNotSupportedError struct {
	config *NativeMessagingConfig
}

func (notSupportErr *ConfigNotSupportedError) Error() string {
	return fmt.Sprintf("%s has unsupported type %s", notSupportErr.config.Executable, notSupportErr.config.Type)
}

type NativeMessagingConfig struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Executable        string   `json:"path"`
	Type              string   `json:"type"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

func (config *NativeMessagingConfig) ParseFile(path string) error {
	var err error
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(config.Executable) {
		config.Executable = filepath.Join(filepath.Dir(path), config.Executable)
	}
	if config.Type != "stdio" {
		return &ConfigNotSupportedError{config}
	}
	return nil
}

func (config *NativeMessagingConfig) WriteFile(path string) error {
	var err error

	pathNew := filepath.Join(filepath.Dir(path), filepath.Base(path)+".new")

	data, err := json.MarshalIndent(*config, "", "    ")
	if err != nil {
		err = fmt.Errorf("creating the new config file failed: %w", err)
		logs.Error(err)
		return err
	}
	err = os.WriteFile(pathNew, data, 0o644)
	if err != nil {
		err = fmt.Errorf("writing the config file to %s failed: %w", pathNew, err)
		logs.Error(err)
		return err
	}
	err = os.Rename(pathNew, path)
	if err != nil {
		err = fmt.Errorf("moving the temporary file %s into position at %s failed: %w", pathNew, path, err)
		logs.Error(err)
		return err
	}
	return nil
}

func (config *NativeMessagingConfig) IsIdentical(configComp *NativeMessagingConfig) bool {
	return reflect.DeepEqual(*config, *configComp)
}

func (config NativeMessagingConfig) CreateCopy() *NativeMessagingConfig {
	allowedExtensions := make([]string, len(config.AllowedExtensions))
	copy(allowedExtensions, config.AllowedExtensions)
	config.AllowedExtensions = allowedExtensions
	return &config
}

func (config *NativeMessagingConfig) ConvertToCustomConfig(browser util.Browser) {
	config.Executable = browser.GetClientPath()
}
