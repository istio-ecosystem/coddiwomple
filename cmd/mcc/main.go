package main

import (
	"fmt"
	"os"

	"github.com/tetratelabs/mcc/cmd"
)

func main() {
	rootCmd := cmd.Root()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
