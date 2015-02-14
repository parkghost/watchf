package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/parkghost/watchf/daemon"
)

const (
	Version         = "0.4.2"
	Program         = "watchf"
	ContinueOnError = false
)

var (
	verbose     bool
	showVersion bool
	stop        bool
	configFile  string
	writeConfig bool

	quit = make(chan os.Signal, 1)
)

func init() {
	flag.BoolVar(&verbose, "V", false, "Show debugging messages")
	flag.BoolVar(&showVersion, "v", false, "Show version and exit")
	flag.BoolVar(&stop, "s", false, "Stop the "+Program+" Daemon (windows is not support)")
	flag.StringVar(&configFile, "f", "."+Program+".conf", "Specifies a configuration file")
	flag.BoolVar(&writeConfig, "w", false, "Write command-line arguments to configuration file (write and exit)")

	flag.Usage = func() {
		command := os.Args[0]
		fmt.Println("Usage:\n  " + command + " [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()

		maxLen := maxLenOfEventName()
		fmt.Println("Events:")
		for _, e := range GeneralEventBits {
			fmt.Printf("  %s  %s\n", PaddingLeft(strings.ToLower(e.Name), maxLen, " "), e.Desc)
		}

		fmt.Printf("Variables:\n"+
			"  %s: The filename of changed file\n"+
			"  %s: The event type of file changes\n",
			VarFilename, VarEventType)

		printExample()
	}
}

func maxLenOfEventName() int {
	maxLenOfName := 0
	for _, event := range GeneralEventBits {
		if maxLenOfName < len(event.Name) {
			maxLenOfName = len(event.Name)
		}
	}
	return maxLenOfName
}

func main() {
	flag.Parse()

	// stop daemon via signal
	if stop {
		stopDaemon()
		return
	}

	config := loadConfig()
	dmon := startDaemon(config)

	waitForStop(dmon)
}

func stopDaemon() {
	dmon := daemon.NewDaemon(Program, nil)
	if err := dmon.Stop(); err != nil {
		fmt.Printf("cannot stop process:%d caused by:\n%s\n", dmon.GetPid(), err)
		os.Exit(-1)
	}
}

func loadConfig() (config *Config) {
	config = GetDefaultConfig()

	if showVersion {
		fmt.Println("version:", Version)
		os.Exit(0)
	}

	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(0)
	}

	Logln("version:", Version)
	Logln("command-line arguments:", os.Args[1:])

	if writeConfig {
		if err := WriteConfigToFile(config); err != nil {
			fmt.Fprintf(os.Stderr, "cannot write configuration file: %v", err)
		} else {
			fmt.Println("the configuration file was saved successfully")
			os.Exit(0)
		}
	}

	if flag.NFlag() == 0 || (flag.NFlag() == 1 && verbose) {
		if newConfig, err := LoadConfigFromFile(); err != nil {
			Logf("cannot load configuration file: %v", err)
		} else {
			config = newConfig
		}
	}
	Logf("configuration: %+v", config)

	if len(config.Commands) == 0 && !stop {
		flag.Usage()
		os.Exit(-1)
	}

	return
}

func startDaemon(config *Config) *daemon.Daemon {
	service, err := NewWatchService(".", config)
	checkError(err)

	dmon := daemon.NewDaemon(Program, service)
	err = dmon.Start()
	checkError(err)

	return dmon
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func waitForStop(daemon *daemon.Daemon) {
	signal.Notify(quit, os.Kill, os.Interrupt)

	<-quit
	if err := daemon.Stop(); err != nil {
		fmt.Printf(Program+" stop failed: %s\n", err)
	} else {
		fmt.Println(Program + " stopped")
	}
}
