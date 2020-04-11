package config

import (
	"github.com/spf13/viper"
)

func init() {
	Config = viper.New()
	Config.SetDefault("GEODB_PORT", ":8080")
	Config.SetDefault("GEODB_PATH", "/tmp/geodb")
	Config.AutomaticEnv()
}

var Config *viper.Viper
