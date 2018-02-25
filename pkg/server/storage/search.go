package storage

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
)

var searchers = []searcher{
	&btreeSearcher{
		searcherName: "PatientEntityID",
		et:           patient,
		indexedValue: entityIDCol,
	},
	&trigramSearcher{
		searcherName: "PatientName",
		et:           patient,
		indexedValue: "(" + strings.Join([]string{
			nonEmptyUpper(lastNameCol),
			nonEmptyUpper(firstNameCol),
		}, " || ' ' || ") + ")",
	},

	&btreeSearcher{
		searcherName: "OfficeEntityID",
		et:           office,
		indexedValue: entityIDCol,
	},
	&trigramSearcher{
		searcherName: "OfficeName",
		et:           office,
		indexedValue: nonEmptyUpper(nameCol),
	},
}

func nonEmptyUpper(colName string) string {
	return fmt.Sprintf("COALESCE(UPPER(%s), '')", colName)
}

type searcher interface {
	entityType() entityType
	name() string
	predicate() string
	similarity() string
	preprocQuery(raw string) string
}

type btreeSearcher struct {
	et            entityType
	searcherName  string
	indexedValue  string
	caseSensitive bool
}

func (ps *btreeSearcher) entityType() entityType {
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
	return fmt.Sprintf("char_length($1)::real / char_length(%s)::real AS %s", entityIDCol,
		similarityCol)
}

func (ps *btreeSearcher) preprocQuery(raw string) string {
	if !ps.caseSensitive {
		return strings.ToUpper(raw) + "%"
	}
	return raw + "%"
}

type trigramSearcher struct {
	et            entityType
	searcherName  string
	indexedValue  string
	caseSensitive bool
}

func (ts *trigramSearcher) entityType() entityType {
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
	merge(rows rows, searchName string, et entityType) error
	top(n uint) entitySims
}

type searchResultMergerImpl struct {
	sims map[string]*entitySim
	mu   sync.Mutex
}

func newSearchResultMerger() searchResultMerger {
	return &searchResultMergerImpl{
		sims: make(map[string]*entitySim),
	}
}

func (srm *searchResultMergerImpl) merge(rs rows, searchName string, et entityType) error {
	defer rs.Close()
	for rs.Next() {

		// prepare the destination slice for the entity with an extra slot for it's
		// similarity, which we assume is the last in the search query
		_, entityDest, createEntity := prepEntityScan(et, 1)
		var simDest float64
		dest := append(entityDest, &simDest)
		if err := rs.Scan(dest...); err != nil {
			return err
		}
		e := createEntity()
		srm.mu.Lock()
		if _, in := srm.sims[e.EntityId]; !in {
			srm.sims[e.EntityId] = newEntitySim(e)
		}
		srm.sims[e.EntityId].add(searchName, simDest)
		srm.mu.Unlock()
	}
	return nil
}

func (srm *searchResultMergerImpl) top(n uint) entitySims {
	ess := &entitySims{}
	heap.Init(ess)
	srm.mu.Lock()
	for _, es := range srm.sims {
		if ess.Len() < int(n) || es.similarity() > ess.Peak().similarity() {
			heap.Push(ess, es)
		}
		if ess.Len() > int(n) {
			heap.Pop(ess)
		}
	}
	srm.mu.Unlock()
	sort.Sort(sort.Reverse(ess)) // sort descending
	return (*ess)[:n]
}

// entitySim contains an *api.Entity and its similarities to the query for a number of different
// searches
type entitySim struct {
	e                  *api.Entity
	searches           []string
	similarities       []float64
	similaritySuffStat float64
}

func newEntitySim(e *api.Entity) *entitySim {
	return &entitySim{
		e:            e,
		searches:     make([]string, 0),
		similarities: make([]float64, 0),
	}
}

func (e *entitySim) add(search string, similarity float64) {
	e.searches = append(e.searches, search)
	e.similarities = append(e.similarities, similarity)
	// L-2 suff stat is sum of squares
	e.similaritySuffStat += similarity * similarity
}

func (e *entitySim) similarity() float64 {
	return math.Sqrt(e.similaritySuffStat)
}

// entitySims is a min-heap of entity similarities
type entitySims []*entitySim

func (es entitySims) Len() int {
	return len(es)
}

func (es entitySims) Less(i, j int) bool {
	return es[i].similarity() < es[j].similarity()
}

func (es entitySims) Swap(i, j int) {
	es[i], es[j] = es[j], es[i]
}
func (es *entitySims) Push(x interface{}) {
	*es = append(*es, x.(*entitySim))
}

func (es *entitySims) Pop() interface{} {
	old := *es
	n := len(old)
	x := old[n-1]
	*es = old[0 : n-1]
	return x
}

func (es entitySims) Peak() *entitySim {
	return es[0]
}
