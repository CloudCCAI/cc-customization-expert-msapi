package main

import (
	"os"

	"cloudcc-customization-expert-go/internal/command"
)

func main() {
	os.Exit(command.Run(os.Args[1:], os.Stdout, os.Stderr, ""))
}
