package main

import (
	"flag"
	"fmt"
	"github.com/parkghost/watchf/daemon"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	Version         = "0.1.7"
	Program         = "watchf"
	ContinueOnError = false
)

var (
	commands    StringSet
	sensitive   time.Duration
	stop        bool
	showVersion bool
	pattern     = "*"
	verbose     bool

	quit = make(chan os.Signal, 1)
)

func init() {

	flag.Var(&commands, "c", "Add arbitrary command (repeatable)")
	flag.DurationVar(&sensitive, "t", time.Duration(500)*time.Millisecond, "The time sensitive for avoid execute command frequently (time unit: ns/us/ms/s/m/h)")
	flag.BoolVar(&stop, "s", false, "To stop the "+Program+" Daemon (windows is not support)")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.BoolVar(&verbose, "V", false, "show debugging message")

	flag.Usage = func() {
		command := os.Args[0]
		fmt.Println("Usage:\n  " + command + " options ['pattern']")
		fmt.Println("Options:")
		flag.PrintDefaults()

		fmt.Println(`Patterns:
  '*'         matches any sequence of non-Separator characters e.g. '*.txt'
  '?'         matches any single non-Separator character       e.g. 'ab?.txt'
  '[' [ '^' ] { character-range } ']'                          e.g. 'ab[b-d].txt'
              character class (must be non-empty)
   c          matches character c (c != '*', '?', '\\', '[')   e.g. 'abc.txt'
Variables:
  $f: The filename of changed file
  $t: The event type of file changes (event type: CREATE/MODIFY/DELETE/RENAME)
  `)

		fmt.Println("Example 1:")
		fmt.Println("  " + command + " -c 'go vet' -c 'go test' -c 'go install' '*.go'")
		fmt.Println("Example 2(Daemon):")
		fmt.Println("  " + command + " -c 'process.sh $f $t' '*.txt' &")
		fmt.Println("  " + command + " -s")
	}

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
	daemon := daemon.NewDaemon(Program, NewWatchService(".", pattern, sensitive, commands))
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

	if len(flag.Args()) > 0 {
		pattern = strings.Trim(strings.Join(flag.Args(), " "), " ")
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
