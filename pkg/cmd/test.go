package cmd

import (
	cerrors "github.com/drausin/libri/libri/common/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	directoriesFlag = "directories"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test one or more directory servers",
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.PersistentFlags().StringSlice(directoriesFlag, nil,
		"space-separated addresses of directory(s)")

	// bind viper flags
	viper.SetEnvPrefix(envVarPrefix) // look for env vars with "DIRECTORY_" prefix
	viper.AutomaticEnv()             // read in environment variables that match
	cerrors.MaybePanic(viper.BindPFlags(testCmd.PersistentFlags()))
}
