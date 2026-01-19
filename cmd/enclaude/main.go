package main

import (
	"os"

	"github.com/jakenelson/enclaude/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
