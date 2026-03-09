package main

import (
	"os"

	"github.com/YumNumm/google-play-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
