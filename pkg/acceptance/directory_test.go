// +build acceptance

package acceptance

import (
	"context"
	"math/rand"
	"net"
	"testing"
	"time"

	errors2 "github.com/drausin/libri/libri/common/errors"
	"github.com/drausin/libri/libri/common/logging"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/postgres/migrations"
	bstorage "github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/mattes/migrate/source/go-bindata"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

type parameters struct {
	nDirectories uint
	nPuts        uint
	nGets        uint
	nSearches    uint
	updateRatio  float32
	searchRatio  float32
	searchLimit  uint32
	rqTimeout    time.Duration
	logLevel     zapcore.Level
}

type state struct {
	rng              *rand.Rand
	dbURL            string
	directories      []*server.Directory
	directoryClients []api.DirectoryClient
	entities         []*api.Entity
	tearDownPostgres func() error
}

func TestAcceptance(t *testing.T) {
	params := &parameters{
		nDirectories: 3,
		nPuts:        64,
		nGets:        64,
		nSearches:    16,
		updateRatio:  0.25,
		searchRatio:  0.75,
		searchLimit:  api.MaxSearchLimit,
		rqTimeout:    3 * time.Second,
		logLevel:     zapcore.InfoLevel,
	}
	st := setUp(t, params)

	testPutNewEntities(t, params, st)

	testPutUpdatedEntities(t, params, st)

	testGetEntities(t, params, st)

	testSearchEntities(t, params, st)

	tearDown(t, st)
}

func testPutNewEntities(t *testing.T, params *parameters, st *state) {
	st.entities = make([]*api.Entity, params.nPuts)

	for i := range st.entities {
		st.entities[i] = CreateTestEntity(st.rng)

		rq := &api.PutEntityRequest{Entity: st.entities[i]}
		ctx, cancel := context.WithTimeout(context.Background(), params.rqTimeout)
		rp, err := st.randClient().PutEntity(ctx, rq)
		cancel()
		assert.Nil(t, err)
		st.entities[i].EntityId = rp.EntityId
	}

}

func testPutUpdatedEntities(t *testing.T, params *parameters, st *state) {
	for i, e := range st.entities {
		if st.rng.Float32() > params.updateRatio {
			continue
		}
		UpdateTestEntity(e)

		rq := &api.PutEntityRequest{Entity: st.entities[i]}
		ctx, cancel := context.WithTimeout(context.Background(), params.rqTimeout)
		rp, err := st.randClient().PutEntity(ctx, rq)
		cancel()
		assert.Nil(t, err)
		assert.Equal(t, e.EntityId, rp.EntityId)
	}
}

func testGetEntities(t *testing.T, params *parameters, st *state) {
	for _, e := range st.entities {
		rq := &api.GetEntityRequest{EntityId: e.EntityId}
		ctx, cancel := context.WithTimeout(context.Background(), params.rqTimeout)
		rp, err := st.randClient().GetEntity(ctx, rq)
		cancel()
		assert.Nil(t, err)
		assert.Equal(t, e, rp.Entity)
	}
}

func testSearchEntities(t *testing.T, params *parameters, st *state) {
	for _, e := range st.entities {
		if st.rng.Float32() > params.searchRatio {
			continue
		}

		rq := &api.SearchEntityRequest{
			Query: GetTestSearchQueryFromEntity(st.rng, e),
			Limit: params.searchLimit,
		}
		ctx, cancel := context.WithTimeout(context.Background(), params.rqTimeout)
		rp, err := st.randClient().SearchEntity(ctx, rq)
		cancel()
		assert.Nil(t, err)
		assert.True(t, len(rp.Entities) > 0)

		// should find entity in results
		found := false
		for _, re := range rp.Entities {
			if re.EntityId == e.EntityId {
				found = true
				break
			}
		}
		assert.True(t, found)
	}
}

func setUp(t *testing.T, params *parameters) *state {
	dbURL, cleanup, err := bstorage.StartTestPostgres()
	if err != nil {
		t.Fatal(err)
	}
	st := &state{
		rng:              rand.New(rand.NewSource(0)),
		dbURL:            dbURL,
		tearDownPostgres: cleanup,
	}
	createAndStartDirectories(params, st)
	return st
}

func createAndStartDirectories(params *parameters, st *state) {
	configs, addrs := newDirectoryConfigs(params, st)
	catalogs := make([]*server.Directory, params.nDirectories)
	directoryClients := make([]api.DirectoryClient, params.nDirectories)
	up := make(chan *server.Directory, 1)

	for i := uint(0); i < params.nDirectories; i++ {
		go func() {
			err := server.Start(configs[i], up)
			errors2.MaybePanic(err)
		}()

		// wait for server to come up
		catalogs[i] = <-up

		// set up client to it
		conn, err := grpc.Dial(addrs[i].String(), grpc.WithInsecure())
		errors2.MaybePanic(err)
		directoryClients[i] = api.NewDirectoryClient(conn)
	}

	st.directories = catalogs
	st.directoryClients = directoryClients
}

func newDirectoryConfigs(params *parameters, st *state) ([]*server.Config, []*net.TCPAddr) {
	startPort := uint(10100)
	configs := make([]*server.Config, params.nDirectories)
	addrs := make([]*net.TCPAddr, params.nDirectories)

	storageParams := storage.NewDefaultParameters()
	storageParams.Type = bstorage.Postgres

	for i := uint(0); i < params.nDirectories; i++ {
		serverPort, metricsPort := startPort+i*10, startPort+i*10+1
		configs[i] = server.NewDefaultConfig().
			WithStorage(storageParams).
			WithDBUrl(st.dbURL)
		configs[i].WithServerPort(uint(serverPort)).
			WithMetricsPort(uint(metricsPort)).
			WithLogLevel(params.logLevel)
		addrs[i] = &net.TCPAddr{IP: net.ParseIP("localhost"), Port: int(serverPort)}
	}
	return configs, addrs
}

func tearDown(t *testing.T, st *state) {
	for _, d := range st.directories {
		d.StopServer()
	}
	logger := &bstorage.ZapLogger{Logger: logging.NewDevInfoLogger()}
	m := bstorage.NewBindataMigrator(
		st.dbURL,
		bindata.Resource(migrations.AssetNames(), migrations.Asset),
		logger,
	)
	err := m.Down()
	assert.Nil(t, err)

	err = st.tearDownPostgres()
	assert.Nil(t, err)
}

func (st *state) randClient() api.DirectoryClient {
	return st.directoryClients[st.rng.Int31n(int32(len(st.directoryClients)))]
}
