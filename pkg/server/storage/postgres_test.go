package storage

import (
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/elxirhealth/directory/pkg/server/storage/migrations"
	"github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/mattes/migrate/source/go-bindata"
	"github.com/stretchr/testify/assert"
)

var (
	setUpPostgresTest func(t *testing.T) (dbURL string, tearDown func() error)
)

func TestMain(m *testing.M) {
	dbURL, cleanup, err := storage.StartTestPostgres()
	if err != nil {
		if err2 := cleanup(); err2 != nil {
			log.Fatal(err2.Error())
		}
		log.Fatal(err.Error())
	}
	setUpPostgresTest = func(t *testing.T) (string, func() error) {
		return dbURL, storage.SetUpTestPostgresDB(t, dbURL, bindata.Resource(
			migrations.AssetNames(),
			func(name string) ([]byte, error) { return migrations.Asset(name) },
		))
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := cleanup(); err != nil {
		log.Fatal(err.Error())
	}

	os.Exit(code)
}

func TestPostgresStorer_PutGetEntity(t *testing.T) {
	dbURL, tearDown := setUpPostgresTest(t)
	defer tearDown()

	rng := rand.New(rand.NewSource(0))
	idGen := NewNaiveIDGenerator(rng, 9)
	s, err := NewPostgresStorer(dbURL, idGen)
	assert.Nil(t, err)
	assert.NotNil(t, s)
}
