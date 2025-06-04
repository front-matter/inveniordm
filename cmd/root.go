/*
Copyright © 2025 Front Matter <info@front-matter.io>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "inveniordm",
	Version: "v0.1.5",
	Short:   "",
	Long:    ``,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("root called")
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("host", "", "", "InvenioRDM host")
	rootCmd.PersistentFlags().StringP("token", "", "", "API token")
}
