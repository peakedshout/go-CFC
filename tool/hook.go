package tool

import (
	"encoding/json"
	"os"
)

func GetCFCHookConfig(path string) *CFCHookConfig {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var config CFCHookConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		panic(err)
	}
	return &config
}

type CFCHookConfig struct {
	Ct struct {
		Key  string `json:"Key"`
		Name string `json:"Name"`
		IP   string `json:"IP"`
		Port string `json:"Port"`
	} `json:"Ct"`
	Tcp struct {
		Server []struct {
			Name string `json:"Name"`
			IP   string `json:"IP"`
			Port string `json:"Port"`
		} `json:"Server"`
		Client []struct {
			Name string `json:"Name"`
			IP   string `json:"IP"`
			Port string `json:"Port"`
		} `json:"Client"`
	} `json:"Tcp"`
}
