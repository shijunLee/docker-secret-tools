package config

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_InitConfig(t *testing.T) {
	var cfgFile = "../../test/config.yaml"
	InitConfig(cfgFile)
	data, err := json.Marshal(GlobalConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(string(data))
}
