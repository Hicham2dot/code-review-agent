package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Code Review Agent - v0.1")
	if len(os.Args) < 2 {
		fmt.Println("Usage: code-review-agent [command] [options]")
		fmt.Println("Commands: analyze, batch, cache-clear")
		os.Exit(1)
	}
}
