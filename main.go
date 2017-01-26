package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// APIConfig declares a configuration nessessary to make API calls to Reddit
type APIConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"access_token"`
	Secret   string `yaml:"client_secret"`
}

var config APIConfig

func main() {
	// Read config from file
	rawConfig, err := ioutil.ReadFile("auth.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		panic(err)
	}
	fmt.Println(config)
}
