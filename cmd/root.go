/*
Copyright © 2021 Google

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
)

const (
	ProjectFlag           = "project"
	ContainerBasePathFlag = "container-base-url"
)

var (
	ProjectID         string
	ContainerBasePath string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "legacynetwork",
	Short: "Migrate GCE Legacy Networks and GKE Clusters to VPC networks.",
	Long:  `Use migrate subcommand.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	pflags := rootCmd.PersistentFlags()
	pflags.StringVarP(&ProjectID, ProjectFlag, "p", ProjectID, "project ID")
	rootCmd.MarkFlagRequired(ProjectFlag)

	pflags.StringVar(&ContainerBasePath, ContainerBasePathFlag, ContainerBasePath, "Custom URL for the container API endpoint (for testing).")
	pflags.MarkHidden(ContainerBasePathFlag)

	// rootCmd.AddCommand(migrateCmd())
}
