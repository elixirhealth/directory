package postgres

import (
	"testing"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/stretchr/testify/assert"
)

func TestSearchResultMergerImpl_merge(t *testing.T) {
	search1, search2 := "search1", "search2"
	n := 4
	rows1 := &fixedOfficeRows{ess: testEntitySims(0.1, search1, n)}
	rows2 := &fixedOfficeRows{ess: testEntitySims(0.2, search2, n)}

	srm := newSearchResultMerger()
	err := srm.merge(rows1, search1, storage.Office)
	assert.Nil(t, err)
	err = srm.merge(rows2, search2, storage.Office)
	assert.Nil(t, err)
	assert.Equal(t, n, len(srm.(*searchResultMergerImpl).sims))
	for _, v := range srm.(*searchResultMergerImpl).sims {
		assert.Equal(t, 2, len(v.Similarities))
	}
}

func testEntitySims(simMult float64, search string, n int) storage.EntitySims {
	ess := make(storage.EntitySims, n)
	for i := range ess {
		sim := simMult * float64(i)
		es := storage.NewEntitySim(api.NewTestOffice(i, true))
		es.Add(search, sim)
		ess[i] = es

	}
	return ess
}

func TestSearchResultMergerImpl_top(t *testing.T) {
	searchName := "Search1"
	es1 := storage.NewEntitySim(&api.Entity{EntityId: "entity1"})
	es1.Add(searchName, 0.1)
	es2 := storage.NewEntitySim(&api.Entity{EntityId: "entity2"})
	es2.Add(searchName, 0.3)
	es3 := storage.NewEntitySim(&api.Entity{EntityId: "entity3"})
	es3.Add(searchName, 0.2)
	es4 := storage.NewEntitySim(&api.Entity{EntityId: "entity4"})
	es4.Add(searchName, 0.4)
	srm := &searchResultMergerImpl{
		sims: map[string]*storage.EntitySim{
			es1.E.EntityId: es1,
			es2.E.EntityId: es2,
			es3.E.EntityId: es3,
			es4.E.EntityId: es4,
		},
	}
	top := srm.top(2)
	assert.Equal(t, 2, len(top))
	assert.Equal(t, storage.EntitySims{es4, es2}, top)
}

type fixedOfficeRows struct {
	ess      storage.EntitySims
	cursor   int
	errErr   error
	closeErr error
}

func (fr *fixedOfficeRows) Err() error {
	return fr.errErr
}

func (fr *fixedOfficeRows) Scan(dest ...interface{}) error {
	e := fr.ess[fr.cursor].E
	f := e.TypeAttributes.(*api.Entity_Office).Office
	dest[0] = &e.EntityId
	dest[1] = &f.Name
	sim := fr.ess[fr.cursor].Similarity()
	dest[2] = &sim
	fr.cursor++
	return nil
}

func (fr *fixedOfficeRows) Next() bool {
	return fr.cursor < len(fr.ess)
}

func (fr *fixedOfficeRows) Close() error {
	return fr.closeErr
}
