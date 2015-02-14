// +build windows

package main

import (
	"fmt"
	"os"
)

func printExample() {
	fmt.Printf(`
Example 1:
  %[1]s -e "write,remove,create" -c "go test" -c "go vet" -include ".go$"
Example 2(with configuration file):
  %[1]s -e "write,remove,create" -c "go test" -c "go vet" -include ".go$" -w
  %[1]s
`, os.Args[0])
}
