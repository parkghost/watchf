package main

import (
	"flag"
	"fmt"
	"github.com/parkghost/watchf/daemon"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"
)

const (
	Version         = "0.2.0"
	Program         = "watchf"
	ContinueOnError = false
)

const (
	VarFilename  = "%f"
	VarEventType = "%t"
)

var (
	commands    StringSet
	events      string
	interval    time.Duration
	stop        bool
	showVersion bool
	recursive   bool
	regexExpr   string
	verbose     bool

	watchFlags uint32
	pattern    *regexp.Regexp
	quit       = make(chan os.Signal, 1)
)

func init() {

	flag.Var(&commands, "c", "Add arbitrary command (repeatable)")
	flag.StringVar(&events, "e", "all", "Listen for specific event(s) (comma separated list)")
	flag.StringVar(&regexExpr, "p", ".*", "File name matches regular expression pattern (perl-style)")
	flag.DurationVar(&interval, "i", time.Duration(0)*time.Millisecond, "The interval limit the frequency of the command executions, if equal to 0, there is no limit (time unit: ns/us/ms/s/m/h)")
	flag.BoolVar(&stop, "s", false, "Stop the "+Program+" Daemon (windows is not support)")
	flag.BoolVar(&recursive, "r", false, "Watch directories recursively")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&verbose, "V", false, "Show debugging messages")

	flag.Usage = func() {
		command := os.Args[0]
		fmt.Println("Usage:\n  " + command + " options")
		fmt.Println("Options:")
		flag.PrintDefaults()

		maxLen := maxLenOfEventName()
		fmt.Println("Events:")
		for _, e := range GeneralEventBits {
			fmt.Printf("  %s  %s\n", paddingStr(strings.ToLower(e.Name), maxLen, " "), e.Desc)
		}

		fmt.Printf("Variables:\n"+
			"  %s: The filename of changed file\n"+
			"  %s: The event type of file changes\n",
			VarFilename, VarEventType)

		showExample()
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

func paddingStr(original string, maxLen int, char string) string {
	return original + strings.Repeat(char, maxLen-len(original))
}

func main() {

	// command line parsing
	parseOptions()

	// stop daemon via signal
	if stop {
		stopDaemon()
		return
	}

	// start daemon
	daemon := daemon.NewDaemon(Program, NewWatchService(".", watchFlags, recursive, pattern, interval, commands))
	checkError(daemon.Start())

	// stop daemon
	waitForStop(daemon)
}

func parseOptions() {
	flag.Parse()

	if showVersion || verbose {
		fmt.Println("version " + Version)
	}

	if verbose {
		log.Println("command-line arguments", os.Args[1:])
	}

	if len(commands) == 0 && !stop {
		flag.Usage()
		os.Exit(-1)
	}

	var err error
	if watchFlags, err = getFlagsValue(events); err != nil {
		log.Println(err)
		flag.Usage()
		os.Exit(-1)
	}

	pattern, err = regexp.Compile(regexExpr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}

func stopDaemon() {
	daemon := daemon.NewDaemon(Program, nil)
	if err := daemon.Stop(); err != nil {
		fmt.Printf("cannot stop process:%d caused by:\n%s\n", daemon.GetPid(), err)
		os.Exit(-1)
	}
}

func checkError(err error) {
	if err != nil {
		log.Println(err)
		close(quit)
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

type StringSet []string

func (f *StringSet) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *StringSet) Set(value string) error {
	*f = append(*f, value)
	return nil
}
