package main

import (
	"flag"
	"fmt"
	"regexp"
	"time"

	"github.com/parkghost/watchf/config"

	log "github.com/Sirupsen/logrus"
)

var defaultConfig = &config.Config{
	Events:         []string{"all"},
	IncludePattern: config.Pattern{Regexp: regexp.MustCompile(".*")},
	ExcludePattern: config.Pattern{Regexp: regexp.MustCompile("^\\.")},
	Commands:       []string{},
}

func init() {
	flag.BoolVar(&defaultConfig.Recursive, "r", false, "Watch directories recursively")
	flag.Var(&defaultConfig.IncludePattern, "include", "Process any events whose file name matches file name matches specified regular expression pattern (perl-style)")
	flag.Var(&defaultConfig.ExcludePattern, "exclude", "Do not process any events whose file name matches specified regular expression pattern (perl-style)")
	flag.DurationVar(&defaultConfig.Interval, "i", 100*time.Millisecond, "The interval limit the frequency of the command executions, if equal to 0, there is no limit (time unit: ns/us/ms/s/m/h)")
	flag.Var(&defaultConfig.Events, "e", "Listen for specific event(s) (comma separated list)")
	flag.Var((*stringSet)(&defaultConfig.Commands), "c", "Add arbitrary command (repeatable)")
}

func loadConfig() (*config.Config, error) {
	if len(defaultConfig.Commands) > 0 {
		return defaultConfig, nil
	}
	log.Debugf("Load configuration from file: %s", configFile)
	return config.FromFile(configFile)
}

type stringSet []string

func (f *stringSet) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *stringSet) Set(value string) error {
	*f = append(*f, value)
	return nil
}
