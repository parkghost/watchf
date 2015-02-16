package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/parkghost/watchf"
	"github.com/parkghost/watchf/config"

	log "github.com/Sirupsen/logrus"
	"github.com/shiena/ansicolor"
	"golang.org/x/net/context"
)

const (
	Version = "0.5"
	Program = "watchf"
)

var (
	verbose     bool
	configFile  string
	writeConfig bool
)

func init() {
	flag.BoolVar(&verbose, "V", false, "Show debugging messages")
	flag.StringVar(&configFile, "f", "."+Program+".conf", "Specifies a configuration file")
	flag.BoolVar(&writeConfig, "w", false, "Write command-line arguments to configuration file (write and exit)")
	flag.Usage = func() {
		fmt.Println("Usage:\n  " + os.Args[0] + " [options]\n")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println(`
Events:
     all  Create/Write/Remove/Rename/Chmod
  create  File/directory created in watched directory
  write   File written in watched directory
  remove  File/directory deleted from watched directory
  rename  File moved out of watched directory
  chmod   File Metadata changed

Variables:
      %f  The filename of changed file
      %t  The event type of file changes`)

		printExample()
	}

	log.SetOutput(ansicolor.NewAnsiColorWriter(os.Stderr))
}

func main() {
	flag.Parse()
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(-1)
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s(%s)", strings.Title(Program), Version)
	log.Debugf("Command-Line: %s", os.Args[1:])

	if writeConfig {
		handleWriteConfig()
		return
	}

	var cfg *config.Config
	var err error
	if cfg, err = loadConfig(); err != nil {
		log.Fatal("Unable to load configuration:", err)
	}
	log.Debugf("Config: %+v", cfg)
	if err = cfg.Validate(); err != nil {
		log.Fatal("Invalid config:", err)
	}

	service, err := watchf.New(context.Background(), cfg, ".", watchf.NewLimitedHandler(cfg))
	if err != nil {
		log.Fatalf("Init WatchService failed: %s", err)
	}
	log.Debug("Starting WatchService")
	err = service.Start()
	if err != nil {
		log.Fatalf("Start WatchService failed: %s", err)
	}
	log.Debug("Started WatchService")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Kill, os.Interrupt)
	<-quit
	log.Debug("Stopping WatchService")
	err = service.Stop()
	if err != nil {
		log.Fatalf("Stop WatchService failed: %s", err)
	}
	log.Debug("Stopped WatchService")
}

func handleWriteConfig() {
	if err := defaultConfig.SaveToFile(configFile); err != nil {
		log.Fatalf("Cannot write configuration to file: %v", err)
	}
	log.Info("The configuration file was saved successfully")
}
