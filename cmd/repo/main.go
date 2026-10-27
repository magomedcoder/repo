package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runServe()
		return
	}

	switch args[0] {
	case "serve":
		runServe()
	case "user":
		if err := runUser(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "user: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  repo                 start HTTP server (default)
  repo serve           start HTTP server
  repo user create     create a user

Examples:
  repo user create -username owner -email owner@localhost -password 'secret123'
`)
}
