package cmd

import (
	"errors"
	"log"
	"os"

	cerrors "github.com/drausin/libri/libri/common/errors"
	"github.com/drausin/libri/libri/common/logging"
	"github.com/elxirhealth/directory/pkg/server"
	bserver "github.com/elxirhealth/service-base/pkg/server"
	bstorage "github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	serverPortFlag      = "serverPort"
	metricsPortFlag     = "metricsPort"
	profilerPortFlag    = "profilerPort"
	profileFlag         = "profile"
	dbURLFlag           = "dbURL"
	storageMemoryFlag   = "storageMemory"
	storagePostgresFlag = "storagePostgres"
)

var (
	errMultipleStorageTypes = errors.New("multiple storage types specified")
	errNoStorageType        = errors.New("no storage type specified")
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

	startCmd.Flags().Bool(storageMemoryFlag, true,
		"use in-memory storage")
	startCmd.Flags().Bool(storagePostgresFlag, false,
		"use Postgres DB storage")
	startCmd.Flags().String(dbURLFlag, "", "Postgres DB URL")

	// bind viper flags
	viper.SetEnvPrefix(envVarPrefix) // look for env vars with "DIRECTORY_" prefix
	viper.AutomaticEnv()             // read in environment variables that match
	cerrors.MaybePanic(viper.BindPFlags(startCmd.Flags()))
}

func getDirectoryConfig() (*server.Config, error) {
	storageType, err := getStorageType()
	if err != nil {
		return nil, err
	}
	c := server.NewDefaultConfig()
	c.WithServerPort(uint(viper.GetInt(serverPortFlag))).
		WithMetricsPort(uint(viper.GetInt(metricsPortFlag))).
		WithProfilerPort(uint(viper.GetInt(profilerPortFlag))).
		WithLogLevel(logging.GetLogLevel(viper.GetString(logLevelFlag))).
		WithProfile(viper.GetBool(profileFlag))

	c.Storage.Type = storageType
	c.WithDBUrl(viper.GetString(dbURLFlag))

	lg := logging.NewDevLogger(c.LogLevel)
	lg.Info("successfully parsed config", zap.Object("config", c))

	return c, nil
}

func getStorageType() (bstorage.Type, error) {
	if viper.GetBool(storageMemoryFlag) && viper.GetBool(storagePostgresFlag) {
		return bstorage.Unspecified, errMultipleStorageTypes
	}
	if viper.GetBool(storageMemoryFlag) {
		return bstorage.Memory, nil
	}
	if viper.GetBool(storagePostgresFlag) {
		return bstorage.Postgres, nil
	}
	return bstorage.Unspecified, errNoStorageType
}
