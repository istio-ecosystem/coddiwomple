package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "mcc",
		Short: "mcc creates mantifests",
		Long:  `Multicloud cross-cluster configuration for Istio`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	rootCmd.AddCommand(
		templateCmd(),
		uiCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
