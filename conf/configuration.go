package conf

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

func init() {

	viper.AddConfigPath("./")
	viper.AddConfigPath("./conf")

	viper.SetConfigName("plugin-config")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, %s\n", err)
		log.Println("Using environment variables only.")
	} else {
		log.Printf("Using config: %s\n", viper.ConfigFileUsed())
	}

	viper.SetEnvPrefix("htplugin")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("plugin.port", "8887")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
}
