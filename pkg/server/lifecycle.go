package server

import (
	"github.com/drausin/libri/libri/common/errors"
	api "github.com/elixirhealth/directory/pkg/directoryapi"
	"github.com/elixirhealth/directory/pkg/server/storage/postgres/migrations"
	bstorage "github.com/elixirhealth/service-base/pkg/server/storage"
	"github.com/mattes/migrate/source/go-bindata"
	"google.golang.org/grpc"
)

// Start starts the server and eviction routines.
func Start(config *Config, up chan *Directory) error {
	d, err := newDirectory(config)
	if err != nil {
		return err
	}

	if err := d.maybeMigrateDB(); err != nil {
		return err
	}

	registerServer := func(s *grpc.Server) { api.RegisterDirectoryServer(s, d) }
	return d.Serve(registerServer, func() { up <- d })
}

// StopServer handles cleanup involved in closing down the server.
func (d *Directory) StopServer() {
	d.BaseServer.StopServer()
	err := d.storer.Close()
	errors.MaybePanic(err)
}

func (d *Directory) maybeMigrateDB() error {
	if d.config.Storage.Type != bstorage.Postgres {
		return nil
	}

	m := bstorage.NewBindataMigrator(
		d.config.DBUrl,
		bindata.Resource(migrations.AssetNames(), migrations.Asset),
		&bstorage.ZapLogger{Logger: d.Logger},
	)
	return m.Up()
}
