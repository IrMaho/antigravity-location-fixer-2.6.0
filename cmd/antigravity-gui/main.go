package main

import (
	"fmt"
	"os"

	"github.com/tawroot/antigravity-cleaner/pkg/gui"
)

func main() {
	if err := gui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Antigravity GUI: %v\n", err)
		os.Exit(1)
	}
}
