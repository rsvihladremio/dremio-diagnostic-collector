package conf

import (
	"fmt"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func ParseConfig(configDir string, supportedExtensions []string, overrides map[string]*pflag.Flag) (foundConfig string) {

	//read viper config
	baseConfig := "ddc"
	viper.SetConfigName(baseConfig) // Name of config file (without extension)
	viper.AddConfigPath(configDir)

	var confFiles []string
	for _, e := range supportedExtensions {
		confFiles = append(confFiles, fmt.Sprintf("%v.%v", baseConfig, e))
	}

	simplelog.Infof("searching in directory %v for the following: %v", configDir, strings.Join(confFiles, ", "))
	//searching for all known
	for _, ext := range supportedExtensions {
		viper.SetConfigType(ext)
		unableToReadConfigError := viper.ReadInConfig()
		if unableToReadConfigError == nil {
			foundConfig = fmt.Sprintf("%v.%v", baseConfig, ext)
			break
		}
	}

	viper.AutomaticEnv() // Automatically read environment variables

	for k, v := range overrides {
		viper.Set(k, v)
	}
	return
}
