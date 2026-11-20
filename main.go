package main

import (
	"fmt"
	"os"
	_ "time/tzdata"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "serve"

	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "serve":
		return serve()
	case "openapi":
		return printOpenAPI(os.Stdout)
	}

	return fmt.Errorf("unknown command %q; expected serve or openapi", command)
}
