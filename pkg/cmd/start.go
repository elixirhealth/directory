package cmd

import (
	"log"
	"os"

	cerrors "github.com/drausin/libri/libri/common/errors"
	"github.com/drausin/libri/libri/common/logging"
	bserver "github.com/elxirhealth/service-base/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/elxirhealth/directory/pkg/server"
)

const (
	serverPortFlag   = "serverPort"
	metricsPortFlag  = "metricsPort"
	profilerPortFlag = "profilerPort"
	profileFlag      = "profile"
	// TODO put other flag constants here
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start a directory server",
	Run: func(cmd *cobra.Command, args []string) {
		writeBanner(os.Stdout)
		config, err := getDirectoryConfig()
		if err != nil {
			log.Fatal(err)
		}
		if err = server.Start(config, make(chan *server.Directory, 1)); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().Uint(serverPortFlag, bserver.DefaultServerPort,
		"port for the main service")
	startCmd.Flags().Uint(metricsPortFlag, bserver.DefaultMetricsPort,
		"port for Prometheus metrics")
	startCmd.Flags().Uint(profilerPortFlag, bserver.DefaultProfilerPort,
		"port for profiler endpoints (when enabled)")
	startCmd.Flags().Bool(profileFlag, bserver.DefaultProfile,
		"whether to enable profiler")

	// bind viper flags
	viper.SetEnvPrefix(envVarPrefix) // look for env vars with "COURIER_" prefix
	viper.AutomaticEnv()             // read in environment variables that match
	cerrors.MaybePanic(viper.BindPFlags(startCmd.Flags()))
}

func getDirectoryConfig() (*server.Config, error) {
	c := server.NewDefaultConfig()
	c.WithServerPort(uint(viper.GetInt(serverPortFlag))).
		WithMetricsPort(uint(viper.GetInt(metricsPortFlag))).
		WithProfilerPort(uint(viper.GetInt(profilerPortFlag))).
		WithLogLevel(logging.GetLogLevel(viper.GetString(logLevelFlag))).
		WithProfile(viper.GetBool(profileFlag))
	// TODO set other config elements here

	return c, nil
}
