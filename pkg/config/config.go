package config

import (
	"encoding/json"
	//"github.com/osrg/gobgp/internal/pkg/config"
	bgpconfig "github.com/osrg/gobgp/pkg/config"
	"io/ioutil"
)

const (
	ConfigFile = "/export/log/bgp_agent/server.toml"
	ConfigFileType = "toml"
)

type BgpConfig struct {

	//*config.BgpConfigSet
	Config *Config
}

type Config struct{
	EtcdEndpoints string
	EtcdCertFile string
	EtcdKeyFile string
	EtcdCaFile string
}

func LoadConfig(filename string) (*Config, error) {

	buff, err := ioutil.ReadFile(filename)
	if err != nil{
		return nil, err
	}
	c := &Config{}
	err = json.Unmarshal(buff, &c)
	if err != nil{
		return nil, err
	}

	return c, nil
}

func LoadConfigFIle(fileName, fileType, agentconfig string)(* BgpConfig, error)  {
	_, err := bgpconfig.ReadConfigFile(fileName, fileType)
	//fmt.Printf("%v %v", bconfig, err)
	config, err := LoadConfig(agentconfig)
	if err != nil{
		return nil, err
	}
	return &BgpConfig{config }, nil
}

