package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
)

func main() {
	// flags definition
	fs := pflag.NewFlagSet("default", pflag.ContinueOnError)
	fs.Int("port", 8000, "HTTP port")

	err := fs.Parse(os.Args[1:])

	switch {

	case err == pflag.ErrHelp:
		os.Exit(0)
	case err != nil:
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
		fs.PrintDefaults()
		os.Exit(2)
	}

	fs.PrintDefaults()
}
