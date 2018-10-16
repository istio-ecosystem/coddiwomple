// Copyright 2018 Tetrate, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cw",
		Short: "Coddiwomple creates manifests for cross-cluster routing",
		Long:  `Coddiwomple, a multi-cloud cross-cluster configuration generator for Istio`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("cannot use cw directly, use a subcommand")
		},
	}

	rootCmd.AddCommand(
		uiCmd(),
		genCmd(),
	)

	return rootCmd
}
