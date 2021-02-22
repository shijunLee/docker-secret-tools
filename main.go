package main

import (
	"github.com/shijunLee/docker-secret-tools/pkg/config"
	"github.com/spf13/pflag"
)

func main()  {
	cfgFile := ""
	pflag.StringVarP(&cfgFile,"config","c","","set the default config file dir")
	pflag.Parse()
	config.InitConfig(cfgFile)
}
