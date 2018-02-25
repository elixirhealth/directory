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

	// DefaultGetQueryTimeout is the default timeout for DB INSERT or UPDATE queries used to in
	// a Storer's GetEntity method.
	DefaultGetQueryTimeout = 2 * time.Second

	DefaultSearchQueryTimeout = 2 * time.Second
)

// Storer stores and retrieves entities.
type Storer interface {

	// PutEntity inserts a new or updates an existing entity (based on E.EntityId) and returns
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

// EntitySim contains an *api.Entity and its Similarities to the query for a number of different
// Searches
type EntitySim struct {
	E                  *api.Entity
	Searches           []string
	Similarities       []float64
	SimilaritySuffStat float64
}

func NewEntitySim(e *api.Entity) *EntitySim {
	return &EntitySim{
		E:            e,
		Searches:     make([]string, 0),
		Similarities: make([]float64, 0),
	}
}

func (e *EntitySim) Add(search string, similarity float64) {
	e.Searches = append(e.Searches, search)
	e.Similarities = append(e.Similarities, similarity)
	// L-2 suff stat is sum of squares
	e.SimilaritySuffStat += similarity * similarity
}

func (e *EntitySim) Similarity() float64 {
	return math.Sqrt(e.SimilaritySuffStat)
}

// EntitySims is a min-heap of entity Similarities
type EntitySims []*EntitySim

func (es EntitySims) Len() int {
	return len(es)
}

func (es EntitySims) Less(i, j int) bool {
	return es[i].Similarity() < es[j].Similarity()
}

func (es EntitySims) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}
func (es *EntitySims) Push(x interface{}) {
	*es = append(*es, x.(*EntitySim))
}

func (es *EntitySims) Pop() interface{} {
	old := *es
	n := len(old)
	x := old[n-1]
	*es = old[0 : n-1]
	return x
}

func (es EntitySims) Peak() *EntitySim {
	return es[0]
}
