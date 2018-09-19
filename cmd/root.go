package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "mcc",
		Short: "mcc creates mantifests",
		Long:  `Multicloud cross-cluster configuration for Istio`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("cannot use mcc directly, use a subcommand")
		},
	}

	rootCmd.AddCommand(
		uiCmd(),
		configGenCmd(),
	)

	return rootCmd
}
