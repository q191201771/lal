package main

import (
	"encoding/json"
	"github.com/q191201771/lal/httpflv"
	"io/ioutil"
)

type Config struct {
	HttpFlv httpflv.Config `json:"httpflv"`
}

func LoadConf(confFile string) (*Config, error) {
	var config Config
	rawContent, err := ioutil.ReadFile(confFile)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(rawContent, &config); err != nil {
		panic(err)
		return nil, err
	}
	return &config, nil
}
