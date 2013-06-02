// +build freebsd openbsd netbsd darwin linux

package main

import (
	"fmt"
	"os"
)

func showExample() {
	command := os.Args[0]
	fmt.Println("Example 1:")
	fmt.Println("  " + command + " -e \"modify,delete\" -c \"go vet\" -c \"go test\" -c \"go install\" -p \"\\.go$\"")
	fmt.Println("Example 2(with custom variable):")
	fmt.Println("  " + command + " -c \"process.sh %f %t\"")
	fmt.Println("Example 3(with daemon):")
	fmt.Println("  " + command + " -r -c \"rsync -aq $SRC $DST\" &")
	fmt.Println("  " + command + " -s")
	fmt.Println("Example 4(with configuration file):")
	fmt.Println("  " + command + " -e \"modify,delete\" -c \"go vet\" -c \"go test\" -c \"go install\" -p \"\\.go$\" -w")
	fmt.Println("  " + command)
}
