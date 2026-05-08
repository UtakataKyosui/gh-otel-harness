package main

import (
	"os"

	"github.com/UtakataKyosui/gh-c2-harness/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
