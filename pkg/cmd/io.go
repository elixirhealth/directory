package cmd

import (
	"context"
	"log"
	"math/rand"
	"time"

	cerrors "github.com/drausin/libri/libri/common/errors"
	"github.com/drausin/libri/libri/common/logging"
	"github.com/drausin/libri/libri/common/parse"
	"github.com/elixirhealth/directory/pkg/acceptance"
	"github.com/elixirhealth/directory/pkg/directoryapi"
	api "github.com/elixirhealth/directory/pkg/directoryapi"
	server2 "github.com/elixirhealth/service-base/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	timeoutFlag   = "timeout"
	nEntitiesFlag = "nEntities"
	nSearchesFlag = "nSearches"

	logEntityID = "entity_id"
	logNResults = "n_results"
)

var ioCmd = &cobra.Command{
	Use:   "io",
	Short: "test input/output of one or more directory servers",
	Run: func(cmd *cobra.Command, args []string) {
		if err := testIO(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	testCmd.AddCommand(ioCmd)

	ioCmd.Flags().Uint(timeoutFlag, 3,
		"timeout (secs) of directory requests")
	ioCmd.Flags().Uint(nEntitiesFlag, 32,
		"number of entities to put into the directory")
	ioCmd.Flags().Uint(nSearchesFlag, 16,
		"number of searches to perform")

	// bind viper flags
	viper.SetEnvPrefix(envVarPrefix) // look for env vars with prefix
	viper.AutomaticEnv()             // read in environment variables that match
	cerrors.MaybePanic(viper.BindPFlags(ioCmd.Flags()))
}

func testIO() error {
	rng := rand.New(rand.NewSource(0))
	logger := logging.NewDevLogger(logging.GetLogLevel(viper.GetString(logLevelFlag)))
	timeout := time.Duration(viper.GetInt(timeoutFlag) * 1e9)
	addrs, err := parse.Addrs(viper.GetStringSlice(directoriesFlag))
	if err != nil {
		return err
	}
	nEntities := uint(viper.GetInt(nEntitiesFlag))
	nSearches := uint(viper.GetInt(nSearchesFlag))

	dialer := server2.NewInsecureDialer()
	directoryClients := make([]directoryapi.DirectoryClient, len(addrs))
	for i, addr := range addrs {
		conn, err2 := dialer.Dial(addr.String())
		if err != nil {
			return err2
		}
		directoryClients[i] = directoryapi.NewDirectoryClient(conn)
	}

	// put entities
	entities := make([]*api.Entity, nEntities)
	for i := range entities {
		entities[i] = acceptance.CreateTestEntity(rng)
		client := directoryClients[rng.Int31n(int32(len(directoryClients)))]
		rq := &api.PutEntityRequest{Entity: entities[i]}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		rp, err := client.PutEntity(ctx, rq)
		cancel()
		if err != nil {
			logger.Error("entity put failed", zap.Error(err))
			return err
		}
		entities[i].EntityId = rp.EntityId
		logger.Info("entity put succeeded", zap.String(logEntityID, rp.EntityId))
	}

	// search entities
	for c := uint(0); c < nSearches; c++ {
		e := entities[rng.Int31n(int32(nEntities))]
		client := directoryClients[rng.Int31n(int32(len(directoryClients)))]
		rq := &api.SearchEntityRequest{
			Query: acceptance.GetTestSearchQueryFromEntity(rng, e),
			Limit: api.MaxSearchLimit,
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		rp, err := client.SearchEntity(ctx, rq)
		cancel()
		if err != nil {
			logger.Error("entity search failed", zap.Error(err))
			return err
		}
		logger.Info("found search results", zap.Int(logNResults, len(rp.Entities)))
	}

	return nil
}
