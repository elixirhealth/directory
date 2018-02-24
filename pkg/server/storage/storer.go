package storage

import (
	"time"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/pkg/errors"
)

var (
	// ErrMissingEntity indicates when an entity is requested with an ID that does not exist.
	ErrMissingEntity = errors.New("not entity with given ID")

	// ErrDupGenEntityID indicates when a newly generated entity ID already exists.
	ErrDupGenEntityID = errors.New("duplicate entity ID generated")

	errUnknownEntityType = errors.New("unknown entity type")
)

const (
	// Unspecified indicates when the storage type is not specified (and thus should take the
	// default value).
	Unspecified Type = iota

	// Postgres indicates storage backed by a Postgres DB.
	Postgres
)

var (
	// DefaultStorage is the default storage type.
	DefaultStorage = Postgres

	// DefaultPutQueryTimeout is the default timeout for DB INSERT or UPDATE queries used to in
	// a Storer's PutEntity method.
	DefaultPutQueryTimeout = 2 * time.Second

	// DefaultGetQueryTimeout is the default timeout for DB INSERT or UPDATE queries used to in
	// a Storer's GetEntity method.
	DefaultGetQueryTimeout = 2 * time.Second

	DefaultSearchQueryTimeout = 2 * time.Second
)

// Storer stores and retrieves entities.
type Storer interface {

	// PutEntity inserts a new or updates an existing entity (based on e.EntityId) and returns
	// the entity ID.
	PutEntity(e *api.Entity) (string, error)

	// GetEntity retrives the entity with the given entityID.
	GetEntity(entityID string) (*api.Entity, error)

	SearchEntity(query string, limit uint) ([]*api.Entity, error)

	// Close handles any necessary cleanup.
	Close() error
}

// Type indicates the storage backend type.
type Type int

func (t Type) String() string {
	switch t {
	case Postgres:
		return "Postgres"
	default:
		return "Unspecified"
	}
}

// Parameters defines the parameters of the Storer.
type Parameters struct {
	Type               Type
	PutQueryTimeout    time.Duration
	GetQueryTimeout    time.Duration
	SearchQueryTimeout time.Duration
}

// NewDefaultParameters returns a *Parameters object with default values.
func NewDefaultParameters() *Parameters {
	return &Parameters{
		Type:               DefaultStorage,
		PutQueryTimeout:    DefaultPutQueryTimeout,
		GetQueryTimeout:    DefaultGetQueryTimeout,
		SearchQueryTimeout: DefaultSearchQueryTimeout,
	}
}
