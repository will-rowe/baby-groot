package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/will-rowe/baby-groot/src/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints current version number and exits",
	Long:  `Prints current version number and exits`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.VERSION)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
