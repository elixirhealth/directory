package server

import (
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/service-base/pkg/server"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// Directory implements the DirectoryServer interface.
type Directory struct {
	*server.BaseServer
	config *Config
	storer storage.Storer
}

// newDirectory creates a new DirectoryServer from the given config.
func newDirectory(config *Config) (*Directory, error) {
	baseServer := server.NewBaseServer(config.BaseConfig)
	storer, err := getStorer(config, baseServer.Logger)
	if err != nil {
		return nil, err
	}
	return &Directory{
		BaseServer: baseServer,
		config:     config,
		storer:     storer,
	}, nil
}

// PutEntity creates a new or updates an existing entity.
func (d *Directory) PutEntity(
	ctx context.Context, rq *api.PutEntityRequest,
) (*api.PutEntityResponse, error) {
	d.Logger.Debug("received PutEntity request", logPutEntityRq(rq)...)
	if err := api.ValidatePutEntityRequest(rq); err != nil {
		return nil, err
	}
	newEntity := rq.Entity.EntityId == ""
	entityID, err := d.storer.PutEntity(rq.Entity)
	if err != nil {
		return nil, err
	}
	rp := &api.PutEntityResponse{EntityId: entityID}
	d.Logger.Info("put entity", logPutEntityRp(rq, rp, newEntity)...)
	return rp, nil
}

// GetEntity returns an existing entity.
func (d *Directory) GetEntity(
	ctx context.Context, rq *api.GetEntityRequest,
) (*api.GetEntityResponse, error) {
	d.Logger.Debug("received GetEntity request", zap.String(logEntityID, rq.EntityId))
	if err := api.ValidateGetEntityRequest(rq); err != nil {
		return nil, err
	}
	e, err := d.storer.GetEntity(rq.EntityId)
	if err != nil {
		return nil, err
	}
	rp := &api.GetEntityResponse{Entity: e}
	d.Logger.Info("got entity", logGetEntityRp(rp)...)
	return rp, nil
}

// SearchEntity finds existing entities matching a query.
func (d *Directory) SearchEntity(
	ctx context.Context, rq *api.SearchEntityRequest,
) (*api.SearchEntityResponse, error) {
	d.Logger.Debug("received SearchEntity request", logSearchEntityRq(rq)...)
	if err := api.ValidateSearchEntityRequest(rq); err != nil {
		return nil, err
	}
	es, err := d.storer.SearchEntity(rq.Query, uint(rq.Limit))
	if err != nil {
		return nil, err
	}
	rp := &api.SearchEntityResponse{Entities: es}
	if len(rp.Entities) == 0 {
		d.Logger.Info("found no entities", logSearchEntityRp(rq, rp)...)
	} else {
		d.Logger.Info("found entities", logSearchEntityRp(rq, rp)...)
	}
	return rp, nil

}
