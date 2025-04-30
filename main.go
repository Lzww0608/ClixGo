package main

import (
	"os"

	"github.com/Lzww0608/ClixGo/cmd/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
