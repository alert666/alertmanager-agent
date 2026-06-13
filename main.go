package main

import (
	"fmt"
	"os"

	"github.com/alert666/alertmanager-agent/internal/infra"
)

func main() {
	if err := infra.NewCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
