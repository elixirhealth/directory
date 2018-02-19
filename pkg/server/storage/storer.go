package storage

import (
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/pkg/errors"
)

var (
	errUnknownEntityType = errors.New("unknown entity type")
)

type Storer interface {
	PutEntity(e *api.Entity) (string, error)
	GetEntity(entityID string) (*api.Entity, error)
	Close() error
}

func validateEntity(e *api.Entity) error {
	// TODO (drausin) check entityID
	switch e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		// TODO (drausin) add patient validation
		return nil
	case *api.Entity_Office:
		// TODO (drausin) add office validation
		return nil
	}
	panic(errUnknownEntityType)
}
