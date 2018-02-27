package server

import (
	"sync"
	"testing"

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

/*
TODO (drausin) enable when StartTestPostgres is a bit more robust
func TestDirectory_maybeMigrateDB(t *testing.T) {
	dbURL, cleanupDB, err := bstorage.StartTestPostgres()
	if err != nil {
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
	err = cleanupDB()
	assert.Nil(t, err)
}
*/
