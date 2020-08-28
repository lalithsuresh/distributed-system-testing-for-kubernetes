// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"kubetorch/ssapasses/collector"
	"kubetorch/ssapasses/tracker"
	"os"
)

type Config struct {
	Pkg     string `yaml:"pkg"`
	Handler string `yaml:"handler"`
}

func config() *Config {
	data, _ := ioutil.ReadFile("config.yaml")
	config := Config{}
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return nil
	}
	return &config
}

func main() {
	config := config()
	if config == nil {
		fmt.Println("user config invalid")
		os.Exit(1)
	}

	fmt.Println("find side effects for", config.Handler, "in", config.Pkg)
	collector := collector.NewCollector(config.Pkg)
	collector.CollectEntryPoints()
	tracker := tracker.NewTracker(collector)
	tracker.TrackEntryPoints(config.Handler)
}
