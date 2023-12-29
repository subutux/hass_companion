package config

import (
	"encoding/json"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/subutux/hass_companion/hass/auth"
	"github.com/subutux/hass_companion/internal/logger"
)

func Load() error {
	config_folder, err := homedir.Expand("~/.config")
	if err != nil {
		return err
	}
	viper.SetConfigName("hass_companion")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(config_folder)
	err = viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		if _, err := os.Stat(config_folder); os.IsNotExist(err) {
			os.MkdirAll(config_folder, os.ModePerm)
		}
		os.Create(path.Join(config_folder, "hass_companion.yaml"))
	}
	err = viper.ReadInConfig()
	if err != nil {
		return err
	}

	logger.I().Info("config loaded", "file", viper.ConfigFileUsed())
	return nil
}

func Set(conf string, value interface{}) {
	viper.Set(conf, value)
	Save()
}

func Get(conf string) string {
	return viper.GetString(conf)
}

func GetStruct(conf string, v interface{}) (interface{}, error) {
	data := viper.GetString(conf)
	err := json.Unmarshal([]byte(data), &v)
	return v, err
}

func Save() error {
	return viper.WriteConfig()
}

func NewCredentialsFromConfig() auth.Credentials {
	return auth.NewCredentials(Get("server"), Get("auth.clientId"), Get("auth.accessToken"), Get("auth.refreshToken"))
}
