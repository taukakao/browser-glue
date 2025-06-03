package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

var ErrNoExtensions error = errors.New("config does not list any extensions")

type ErrUnsupportedConfigType struct {
	config *NativeMessagingConfig
}

func (notSupportErr *ErrUnsupportedConfigType) Error() string {
	return fmt.Sprintf("%s has unsupported type %s", notSupportErr.config.Executable, notSupportErr.config.Type)
}

type NativeMessagingConfig struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Executable        string   `json:"path"`
	Type              string   `json:"type"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`
}

func (config *NativeMessagingConfig) GetExtensions() []string {
	if len(config.AllowedExtensions) > len(config.AllowedOrigins) {
		return config.AllowedExtensions
	} else {
		return config.AllowedOrigins
	}
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
		return &ErrUnsupportedConfigType{config}
	}
	if len(config.AllowedExtensions) == 0 && len(config.AllowedOrigins) == 0 {
		return ErrNoExtensions
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

	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		logs.Warn("could not create native messaging config folder", err)
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
	allowedOrigins := make([]string, len(config.AllowedOrigins))
	copy(allowedExtensions, config.AllowedExtensions)
	copy(allowedOrigins, config.AllowedOrigins)
	config.AllowedExtensions = allowedExtensions
	config.AllowedOrigins = allowedOrigins
	return &config
}

func (config *NativeMessagingConfig) ConvertToCustomConfig(browser util.Browser) {
	config.Executable = browser.GetClientPath()
}
