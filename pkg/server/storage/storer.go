package storage

import (
	"math"
	"time"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/pkg/errors"
)

var (
	// ErrMissingEntity indicates when an entity is requested with an ID that does not exist.
	ErrMissingEntity = errors.New("not entity with given ID")

	// ErrDupGenEntityID indicates when a newly generated entity ID already exists.
	ErrDupGenEntityID = errors.New("duplicate entity ID generated")

	// ErrUnknownEntityType indicates when the entity type is unknown (usually used in default
	// case of switch statement).
	ErrUnknownEntityType = errors.New("unknown entity type")
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

	// DefaultGetQueryTimeout is the default timeout for DB SELECT queries used to in
	// a Storer's GetEntity method.
	DefaultGetQueryTimeout = 2 * time.Second

	// DefaultSearchQueryTimeout is the default timeout for DB SELECT queries used to in
	// a Storer's SearchEntity method.
	DefaultSearchQueryTimeout = 2 * time.Second
)

// Storer stores and retrieves entities.
type Storer interface {

	// PutEntity inserts a new or updates an existing entity (based on E.EntityId) and returns
	// the entity ID.
	PutEntity(e *api.Entity) (string, error)

	// GetEntity retrives the entity with the given entityID.
	GetEntity(entityID string) (*api.Entity, error)

	// SearchEntity finds {{ limiit }} entities matching the given query, ordered most similar
	// to least.
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

// EntitySim contains an *api.Entity and its Similarities to the query for a number of different
// Searches
type EntitySim struct {
	E                  *api.Entity
	Similarities       map[string]float64
	similaritySuffStat float64
}

// NewEntitySim creates a new *EntitySim for the given *Entity.
func NewEntitySim(e *api.Entity) *EntitySim {
	return &EntitySim{
		E:            e,
		Similarities: make(map[string]float64),
	}
}

// Add adds a new [0, 1] similarity score for the given search name.
func (e *EntitySim) Add(search string, similarity float64) {
	e.Similarities[search] = similarity
	// L-2 suff stat is sum of squares
	e.similaritySuffStat += similarity * similarity
}

// Similarity returns the combined similarity over all the searches.
func (e *EntitySim) Similarity() float64 {
	return math.Sqrt(e.similaritySuffStat)
}

// EntitySims is a min-heap of entity Similarities
type EntitySims []*EntitySim

// Len returns the number of entity sims.
func (es EntitySims) Len() int {
	return len(es)
}

// Less returns whether entity sim i has a similarity less than that of j.
func (es EntitySims) Less(i, j int) bool {
	return es[i].Similarity() < es[j].Similarity()
}

// Swap swaps the entity sim i and j.
func (es EntitySims) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}

// Push adds the given EntitySim to the heap.
func (es *EntitySims) Push(x interface{}) {
	*es = append(*es, x.(*EntitySim))
}

// Pop removes the EntitySim from the root of the heap.
func (es *EntitySims) Pop() interface{} {
	old := *es
	n := len(old)
	x := old[n-1]
	*es = old[0 : n-1]
	return x
}

// Peak returns the EntitySim from the root of the heap.
func (es EntitySims) Peak() *EntitySim {
	return es[0]
}
