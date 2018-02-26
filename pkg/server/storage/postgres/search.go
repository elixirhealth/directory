package postgres

import (
	"container/heap"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/elxirhealth/directory/pkg/server/storage"
)

var searchers = []searcher{
	&btreeSearcher{
		searcherName: "PatientEntityID",
		et:           storage.Patient,
		indexedValue: entityIDCol,
	},
	&trigramSearcher{
		searcherName: "PatientName",
		et:           storage.Patient,
		indexedValue: nonEmptyUpper(lastNameCol, firstNameCol),
	},

	&btreeSearcher{
		searcherName: "OfficeEntityID",
		et:           storage.Office,
		indexedValue: entityIDCol,
	},
	&trigramSearcher{
		searcherName: "OfficeName",
		et:           storage.Office,
		indexedValue: nonEmptyUpper(nameCol),
	},
}

func nonEmptyUpper(cols ...string) string {
	return "(" + strings.Join(cols, " || ' ' || ") + ")"
}

type searcher interface {
	entityType() storage.EntityType
	name() string
	predicate() string
	similarity() string
	preprocQuery(raw string) string
}

type btreeSearcher struct {
	et            storage.EntityType
	searcherName  string
	indexedValue  string
	caseSensitive bool
}

func (ps *btreeSearcher) entityType() storage.EntityType {
	return ps.et
}

func (ps *btreeSearcher) name() string {
	return ps.searcherName
}

func (ps *btreeSearcher) predicate() string {
	return ps.indexedValue + " LIKE $1"
}

func (ps *btreeSearcher) similarity() string {
	// since we assume that match occurred, the similarity is the fraction of indexed
	// indexedValue that the prefix matches
	return fmt.Sprintf("(char_length($1)-1)::real / char_length(%s)::real AS %s", entityIDCol,
		similarityCol)
}

func (ps *btreeSearcher) preprocQuery(raw string) string {
	if !ps.caseSensitive {
		return strings.ToUpper(raw) + "%"
	}
	return raw + "%"
}

type trigramSearcher struct {
	et            storage.EntityType
	searcherName  string
	indexedValue  string
	caseSensitive bool
}

func (ts *trigramSearcher) entityType() storage.EntityType {
	return ts.et
}

func (ts *trigramSearcher) name() string {
	return ts.searcherName
}
func (ts *trigramSearcher) predicate() string {
	return ts.indexedValue + " % $1"
}

func (ts *trigramSearcher) similarity() string {
	return fmt.Sprintf("similarity(%s, $1) AS %s", ts.indexedValue, similarityCol)
}

func (ts *trigramSearcher) preprocQuery(raw string) string {
	if !ts.caseSensitive {
		return strings.ToUpper(raw)
	}
	return raw
}

type searchResultMerger interface {
	merge(rows queryRows, searchName string, et storage.EntityType) (int, error)
	top(n uint) storage.EntitySims
}

type searchResultMergerImpl struct {
	sims map[string]*storage.EntitySim
	mu   sync.Mutex
}

func newSearchResultMerger() searchResultMerger {
	return &searchResultMergerImpl{
		sims: make(map[string]*storage.EntitySim),
	}
}

func (srm *searchResultMergerImpl) merge(
	rs queryRows, searchName string, et storage.EntityType,
) (int, error) {
	n := 0
	for rs.Next() {

		// prepare the destination slice for the entity with an extra slot for it's
		// similarity, which we assume is the last in the search query
		_, entityDest, createEntity := prepEntityScan(et, 1)
		var simDest float32
		dest := append(entityDest, &simDest)
		if err := rs.Scan(dest...); err != nil {
			return 0, err
		}
		e := createEntity()
		srm.mu.Lock()
		if _, in := srm.sims[e.EntityId]; !in {
			srm.sims[e.EntityId] = storage.NewEntitySim(e)
		}
		srm.sims[e.EntityId].Add(searchName, simDest)
		n++
		srm.mu.Unlock()
	}
	return n, rs.Close()
}

func (srm *searchResultMergerImpl) top(n uint) storage.EntitySims {
	ess := &storage.EntitySims{}
	heap.Init(ess)
	srm.mu.Lock()
	for _, es := range srm.sims {
		if ess.Len() < int(n) || es.Similarity() > ess.Peak().Similarity() {
			heap.Push(ess, es)
		}
		if ess.Len() > int(n) {
			heap.Pop(ess)
		}
	}
	srm.mu.Unlock()
	sort.Sort(sort.Reverse(ess)) // sort descending
	if ess.Len() < int(n) {
		return *ess
	}
	return (*ess)[:n]
}
