package server

import (
	"sync"
	"testing"

	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/postgres/migrations"
	bstorage "github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/mattes/migrate/source/go-bindata"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	up := make(chan *Directory, 1)
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	go func(wg2 *sync.WaitGroup) {
		defer wg2.Done()
		err := Start(NewDefaultConfig(), up)
		assert.Nil(t, err)
	}(wg1)

	x := <-up
	assert.NotNil(t, x)

	x.StopServer()
	wg1.Wait()
}

func TestDirectory_maybeMigrateDB(t *testing.T) {
	dbURL, cleanupDB, err := bstorage.StartTestPostgres()
	if err != nil {
		if err2 := cleanupDB(); err2 != nil {
			t.Fatal("test postgres cleanupDB error: " + err2.Error())
		}
		t.Fatal("test postgres start error: " + err.Error())
	}

	cfg := NewDefaultConfig().WithDBUrl(dbURL)
	cfg.Storage.Type = storage.Postgres

	d, err := newDirectory(cfg)
	assert.Nil(t, err)

	err = d.maybeMigrateDB()
	assert.Nil(t, err)

	// cleanup
	m := migrations.NewBindataMigrator(
		dbURL,
		bindata.Resource(migrations.AssetNames(), migrations.Asset),
		&migrations.ZapLogger{Logger: d.Logger},
	)
	err = m.Down()
	assert.Nil(t, err)
	cleanupDB()

}
