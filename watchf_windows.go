// +build windows

package main

import (
	"fmt"
	"os"
)

func showExample() {
	command := os.Args[0]
	fmt.Println("Example 1:")
	fmt.Println("  " + command + " -e \"modify,delete\" -c \"go vet\" -c \"go test\" -c \"go install\" -p \"\\.go$\"")
}
