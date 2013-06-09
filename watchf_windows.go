// +build windows

package main

import (
	"fmt"
	"os"
)

func printExample() {
	command := os.Args[0]
	fmt.Println("Example 1:")
	fmt.Println("  " + command + " -e \"modify,delete\" -c \"go vet\" -c \"go test\" -c \"go install\" -p \"\\.go$\"")
	fmt.Println("Example 2(with configuration file):")
	fmt.Println("  " + command + " -e \"modify,delete\" -c \"go vet\" -c \"go test\" -c \"go install\" -p \"\\.go$\" -w")
	fmt.Println("  " + command)
}
