package server

import (
	"github.com/drausin/libri/libri/common/errors"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"google.golang.org/grpc"
)

// Start starts the server and eviction routines.
func Start(config *Config, up chan *Directory) error {
	c, err := newDirectory(config)
	if err != nil {
		return err
	}

	registerServer := func(s *grpc.Server) { api.RegisterDirectoryServer(s, c) }
	return c.Serve(registerServer, func() { up <- c })
}

// StopServer handles cleanup involved in closing down the server.
func (d *Directory) StopServer() {
	d.BaseServer.StopServer()
	err := d.storer.Close()
	errors.MaybePanic(err)
}
