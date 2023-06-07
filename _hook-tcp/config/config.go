package config

import (
	"encoding/json"
	"github.com/peakedshout/go-CFC/loger"
	"os"
)

type CFCHookConfig struct {
	ProxyServerHost ProxyServerHostConfig `json:"ProxyServerHost"`
	ProxyDeviceBox  ProxyDeviceBoxConfig  `json:"ProxyDeviceBox"`
	Setting         SettingConfig         `json:"Setting"`
}
type ProxyServerHostConfig struct {
	ProxyServerAddr  string `json:"ProxyServerAddr"`
	LinkProxyKey     string `json:"LinkProxyKey"`
	SwitchVPNProxy   bool   `json:"SwitchVPNProxy"`
	SwitchLinkClient bool   `json:"SwitchLinkClient"`
	SwitchUdpP2P     bool   `json:"SwitchUdpP2P"`
}

type ProxyDeviceBoxConfig struct {
	ProxyTcp ProxyTcp `json:"ProxyTcp"`
}
type ProxyTcp struct {
	Server []ProxyTcpServerConfig `json:"Server"`
	Client []ProxyTcpClientConfig `json:"Client"`
}

type ProxyTcpServerConfig struct {
	ListenProxyName string `json:"ListenProxyName"`
	ServerDialAddr  string `json:"ServerDialAddr"`
	ProxyCryptoKey  string `json:"ProxyCryptoKey"`
}

type ProxyTcpClientConfig struct {
	DialProxyName    string `json:"DialProxyName"`
	ClientListenAddr string `json:"ClientListenAddr"`
	ProxyCryptoKey   string `json:"ProxyCryptoKey"`
}

type SettingConfig struct {
	ReLinkTime string `json:"ReLinkTime"`
	LogLevel   uint8  `json:"LogLevel"`
	LogStack   bool   `json:"LogStack"`
}

func ReadConfig(path string) *CFCHookConfig {
	b, err := os.ReadFile(path)
	if err != nil {
		loger.SetLogError(err)
	}
	var config CFCHookConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		loger.SetLogError(err)
	}
	return &config
}
