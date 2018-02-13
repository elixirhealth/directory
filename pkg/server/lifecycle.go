package server

import (
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"google.golang.org/grpc"
)

// Start starts the server and eviction routines.
func Start(config *Config, up chan *Directory) error {
	c, err := newDirectory(config)
	if err != nil {
		return err
	}

	// start Directory aux routines
	// TODO add go x.auxRoutine() or delete comment

	registerServer := func(s *grpc.Server) { api.RegisterDirectoryServer(s, c) }
	return c.Serve(registerServer, func() { up <- c })
}
