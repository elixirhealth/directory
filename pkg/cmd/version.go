package cmd

import (
	"os"

	"github.com/elixirhealth/directory/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the directory version",
	Long:  "print the directory version",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := os.Stdout.WriteString(version.Current.Version.String() + "\n")
		return err
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
