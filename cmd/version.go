package cmd

import (
	"fmt"

	goVersion "github.com/christopherhein/go-version"
	"github.com/spf13/cobra"
)

var (
	shortened = false
	version   string
	commit    string
	date      string
)

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Version will output the current build information",
		Long:  ``,
		Run: func(_ *cobra.Command, _ []string) {
			var response string
			versionOutput := goVersion.New(version, commit, date)

			if shortened {
				response = versionOutput.ToShortened()
			} else {
				response = versionOutput.ToJSON()
			}
			fmt.Printf("%+v", response)
			return
		},
	}

	cmd.Flags().BoolVarP(&shortened, "short", "s", false, "Use shortened output for version information.")

	return cmd
}
