package memory

import (
	"container/heap"
	"sort"
	"strings"
	"sync"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/id"
	"go.uber.org/zap"
)

type storer struct {
	params *storage.Parameters
	idGen  id.Generator
	stored map[string]*api.Entity
	mu     sync.Mutex
	logger *zap.Logger
}

// New creates a new Storer backed by an in-memory map with the given id.Generator, params, and
// logger.
func New(idGen id.Generator, params *storage.Parameters, logger *zap.Logger) storage.Storer {
	return &storer{
		params: params,
		idGen:  idGen,
		stored: make(map[string]*api.Entity),
		logger: logger,
	}
}

func (s *storer) PutEntity(e *api.Entity) (string, error) {
	insert := true
	if e.EntityId != "" {
		insert = false
		if err := s.idGen.Check(e.EntityId); err != nil {
			return "", err
		}
	}
	if err := api.ValidateEntity(e); err != nil {
		return "", err
	}
	if _, err := storage.MaybeAddEntityID(e, s.idGen); err != nil {
		return "", err
	}
	s.mu.Lock()
	if _, in := s.stored[e.EntityId]; in && insert {
		return "", storage.ErrDupGenEntityID
	}
	s.stored[e.EntityId] = e
	s.mu.Unlock()
	s.logger.Debug("successfully stored entity", logPutResult(e.EntityId, insert)...)
	return e.EntityId, nil
}

func (s *storer) GetEntity(entityID string) (*api.Entity, error) {
	if err := s.idGen.Check(entityID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	e, in := s.stored[entityID]
	s.mu.Unlock()
	if !in {
		return nil, storage.ErrMissingEntity
	}
	s.logger.Debug("successfully found entity", zap.String(logEntityID, entityID))
	return e, nil
}

func (s *storer) SearchEntity(query string, limit uint) ([]*api.Entity, error) {
	if err := api.ValidateSearchQuery(query, uint32(limit)); err != nil {
		return nil, err
	}
	s.mu.Lock()
	ess := &storage.EntitySims{}
	heap.Init(ess)

	// just loop through all entities once
	for _, e := range s.stored {
		if matches, searcher, sim := checkMatchesQuery(e, query); matches {
			es := storage.NewEntitySim(e)
			es.Add(searcher, sim)
			if ess.Len() < int(limit) || es.Similarity() > ess.Peak().Similarity() {
				heap.Push(ess, es)
			}
			if ess.Len() > int(limit) {
				heap.Pop(ess)
			}
		}
	}
	s.mu.Unlock()

	sort.Sort(sort.Reverse(ess)) // sort descending
	result := make([]*api.Entity, 0, limit)
	s.logger.Debug("ranked search results", logSearchRanked(query, limit, *ess)...)
	for _, es := range *ess {
		result = append(result, es.E)
	}
	return result, nil
}

func (s *storer) Close() error {
	return nil
}

// checkMatchesQuery checks whether an entity matches a given query under various searcher,
// returning the first match it finds, if any
func checkMatchesQuery(e *api.Entity, query string) (matches bool, searcher string, sim float32) {
	if matches, sim = matchesUpper(query, e.EntityId); matches {
		return true, "EntityID", sim
	}
	switch ta := e.TypeAttributes.(type) {
	case *api.Entity_Patient:
		p := ta.Patient
		if matches, sim = matchesUpper(query, p.LastName, p.FirstName); matches {
			return true, "PatientName", sim
		}
		if matches, sim = matchesUpper(query, p.FirstName, p.LastName); matches {
			return true, "PatientName", sim
		}
	case *api.Entity_Office:
		f := ta.Office
		if matches, sim = matchesUpper(query, f.Name); matches {
			return true, "OfficeName", sim
		}
	}
	return false, "", 0
}

func matchesUpper(query string, vals ...string) (matches bool, sim float32) {
	concatVals := strings.ToUpper(strings.Join(vals, " "))
	if strings.Contains(concatVals, strings.ToUpper(query)) {
		return true, float32(len(query)) / float32(len(concatVals))
	}
	return false, 0
}
