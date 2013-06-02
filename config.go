package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

var (
	defaultConfig = &Config{Version: Version, Events: []string{"all"}, Commands: []string{}}
)

type Config struct {
	Recursive      bool
	Events         CommaStringSet
	IncludePattern string
	Commands       StringSet
	Interval       time.Duration
	Version        string
}

func init() {
	flag.BoolVar(&defaultConfig.Recursive, "r", false, "Watch directories recursively")
	flag.StringVar(&defaultConfig.IncludePattern, "p", ".*", "File name matches regular expression pattern (perl-style)")
	flag.DurationVar(&defaultConfig.Interval, "i", time.Duration(0)*time.Millisecond, "The interval limit the frequency of the command executions, if equal to 0, there is no limit (time unit: ns/us/ms/s/m/h)")
	flag.Var(&defaultConfig.Events, "e", "Listen for specific event(s) (comma separated list)")
	flag.Var(&defaultConfig.Commands, "c", "Add arbitrary command (repeatable)")
}

func GetDefaultConfig() *Config {
	return defaultConfig
}

func WriteConfigToFile(config *Config) (err error) {
	rawdata, err := json.MarshalIndent(&config, "", "	")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(configFile, rawdata, 0644)
	return
}

func LoadConfigFromFile() (newConfig *Config, err error) {
	// TODO: check compatibility
	newConfig = &Config{}
	rawdata, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(rawdata, newConfig)
	return
}

type StringSet []string

func (f *StringSet) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *StringSet) Set(value string) error {
	*f = append(*f, value)
	return nil
}

type CommaStringSet []string

func (f *CommaStringSet) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *CommaStringSet) Set(value string) error {
	*f = strings.Split(strings.Replace(value, " ", "", -1), ",")
	return nil
}
