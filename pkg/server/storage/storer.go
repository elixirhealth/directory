package storage

import (
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/pkg/errors"
)

var (
	ErrMissingEntity     = errors.New("not entity with given ID")
	ErrDupGenEntityID    = errors.New("duplicate entity ID generated")
	errUnknownEntityType = errors.New("unknown entity type")
)

// Storer stores and retrieves entities.
type Storer interface {

	// PutEntity inserts a new or updates an existing entity (based on e.EntityId) and returns
	// the entity ID.
	PutEntity(e *api.Entity) (string, error)

	// GetEntity retrives the entity with the given entityID.
	GetEntity(entityID string) (*api.Entity, error)

	// Close handles any necessary cleanup.
	Close() error
}