package main

import (
	"log"
	"strings"
)

func PaddingLeft(original string, maxLen int, char string) string {
	if n := maxLen - len(original); n > 0 {
		return strings.Repeat(char, n) + original
	}
	return original
}

func Logln(v ...interface{}) {
	if verbose {
		log.Println(v...)
	}
}

func Logf(format string, args ...interface{}) {
	if verbose {
		log.Printf(format, args...)
	}
}
