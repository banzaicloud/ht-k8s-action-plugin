package conf

import (
	"github.com/spf13/viper"
	"fmt"
	"strings"
	"log"
)

func init() {

	viper.AddConfigPath("./")
	viper.AddConfigPath("./conf")

	viper.SetConfigName("plugin-config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	viper.SetEnvPrefix("htplugin")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("plugin.port", "8887")
}

