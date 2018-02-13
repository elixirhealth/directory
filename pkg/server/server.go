package server

import (
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/service-base/pkg/server"
	"golang.org/x/net/context"
)

// Directory implements the DirectoryServer interface.
type Directory struct {
	*server.BaseServer
	config *Config

	// TODO add other things here
}

// newDirectory creates a new DirectoryServer from the given config.
func newDirectory(config *Config) (*Directory, error) {
	baseServer := server.NewBaseServer(config.BaseConfig)

	// TODO add other init

	return &Directory{
		BaseServer: baseServer,
		config:     config,
		// TODO add other things
	}, nil
}

func (x *Directory) PutEntity(
	ctx context.Context, rq *api.PutEntityRequest,
) (*api.PutEntityResponse, error) {
	panic("implement me")
}
