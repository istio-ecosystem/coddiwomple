package cmd

import (
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
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
		configGenCmd(),
	)

	return rootCmd
}
