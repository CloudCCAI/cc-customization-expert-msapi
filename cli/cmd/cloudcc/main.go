package main

import (
	"os"

	"cloudcc-customization-expert-go/internal/command"
	"cloudcc-customization-expert-go/internal/console"
)

func main() {
	os.Exit(command.Run(os.Args[1:], console.Stdout(), console.Stderr(), ""))
}
