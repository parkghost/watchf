package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Recursive      bool
	Events         EventSet
	IncludePattern Pattern
	ExcludePattern Pattern
	Commands       []string
	Interval       time.Duration
}

func (c *Config) SaveToFile(file string) error {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, b, 0644)
}

func (c *Config) Validate() error {
	if len(c.Commands) == 0 {
		return errors.New("require commands")
	}

	if len(c.Events) == 0 {
		return errors.New("require events")
	}
	return nil
}

func FromFile(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := new(Config)
	dec := json.NewDecoder(f)
	err = dec.Decode(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

type EventSet []string

func (es *EventSet) String() string {
	return fmt.Sprint([]string(*es))
}

func (es *EventSet) Set(value string) error {
	*es = make([]string, 0)
	events := strings.Split(value, ",")
	for i := 0; i < len(events); i++ {
		evt := strings.ToLower(strings.TrimSpace(events[i]))
		if evt == "" {
			continue
		}
		if !strings.Contains("all,create,write,remove,rename,chmod", evt) {
			return fmt.Errorf("invalid event: %s", events[i])
		}
		*es = append(*es, evt)
	}
	return nil
}

type Pattern struct {
	*regexp.Regexp
}

func (p *Pattern) String() string {
	return "\"" + p.Regexp.String() + "\""
}

func (p *Pattern) Set(value string) error {
	re, err := regexp.Compile(value)
	if err != nil {
		return err
	}
	p.Regexp = re
	return nil
}

func (p *Pattern) UnmarshalJSON(data []byte) error {
	txt, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	return p.Set(txt)
}

func (p *Pattern) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(p.Regexp.String())), nil
}
