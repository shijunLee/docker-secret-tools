package config

import (
	"fmt"
	"os"
	"path"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

type SetMethod string

var (
	SetMethodWebHook SetMethod = "WebHook"
	SetMethodUpdate  SetMethod = "Update"
)

type Config struct {
	WatchNamespaces   []string  `json:"watchNamespaces" mapstructure:"watchNamespaces"`
	DockerSecretNames []string  `json:"dockerSecretNames" mapstructure:"dockerSecretNames"`
	SetMethod         SetMethod `json:"setMethod" mapstructure:"setMethod"`
	NotManagerOwners  []string  `json:"notManagerOwners" mapstructure:"notManagerOwners"`
	ServerPort        int       `json:"serverPort" mapstructure:"serverPort"`
	ServiceName       string    `json:"serviceName" mapstructure:"serviceName"`
	AutoTLS           bool      `json:"autoTLS" mapstructure:"autoTLS"`
	CertFile          string    `json:"certFile" mapstructure:"certFile"`
	PrivateKeyFile    string    `json:"privateKeyFile" mapstructure:"privateKeyFile"`
	RootCA            string    `json:"rootCA" mapstructure:"rootCA"`
}

var GlobalConfig = &Config{}

func InitConfig(cfgFile string) {
	viper.SetDefault("setMethod", "WebHook")
	viper.SetDefault("serverPort", 8888)
	viper.SetDefault("autoTLS", true)
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".dockerctl" (without extension).
		viper.AddConfigPath(path.Join(home, ".secretool"))
		viper.AddConfigPath("/etc/secretool")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		err := viper.Unmarshal(GlobalConfig)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}
}
