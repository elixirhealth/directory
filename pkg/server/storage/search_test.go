package storage

import (
	"container/heap"
	"fmt"
	"math"
	"testing"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/stretchr/testify/assert"
)

func TestSearchResultMergerImpl_merge(t *testing.T) {
	search1, search2 := "search1", "search2"
	n := 4
	rows1 := &fixedOfficeRows{ess: testEntitySims(0.1, search1, n)}
	rows2 := &fixedOfficeRows{ess: testEntitySims(0.2, search2, n)}

	srm := newSearchResultMerger()
	err := srm.merge(rows1, search1, office)
	assert.Nil(t, err)
	err = srm.merge(rows2, search2, office)
	assert.Nil(t, err)
	assert.Equal(t, n, len(srm.(*searchResultMergerImpl).sims))
	for _, v := range srm.(*searchResultMergerImpl).sims {
		assert.Equal(t, 2, len(v.similarities))
		assert.Equal(t, 2, len(v.searches))
	}
}

func testEntitySims(simMult float64, search string, n int) entitySims {
	ess := make(entitySims, n)
	for i := range ess {
		sim := simMult * float64(i)
		ess[i] = &entitySim{
			similarities:       []float64{sim},
			similaritySuffStat: sim * sim,
			searches:           []string{search},
			e: &api.Entity{
				EntityId: fmt.Sprintf("%d", i),
				TypeAttributes: &api.Entity_Office{
					Office: &api.Office{
						Name: fmt.Sprintf("%s office %d", search, i),
					},
				},
			},
		}

	}
	return ess
}

func TestSearchResultMergerImpl_top(t *testing.T) {
	searchName := "Search1"
	es1 := newEntitySim(&api.Entity{EntityId: "entity1"})
	es1.add(searchName, 0.1)
	es2 := newEntitySim(&api.Entity{EntityId: "entity2"})
	es2.add(searchName, 0.3)
	es3 := newEntitySim(&api.Entity{EntityId: "entity3"})
	es3.add(searchName, 0.2)
	es4 := newEntitySim(&api.Entity{EntityId: "entity4"})
	es4.add(searchName, 0.4)
	srm := &searchResultMergerImpl{
		sims: map[string]*entitySim{
			es1.e.EntityId: es1,
			es2.e.EntityId: es2,
			es3.e.EntityId: es3,
			es4.e.EntityId: es4,
		},
	}
	top := srm.top(2)
	assert.Equal(t, 2, len(top))
	assert.Equal(t, entitySims{es4, es2}, top)
}

func TestEntitySims(t *testing.T) {
	searchName := "Search1"
	es1 := newEntitySim(&api.Entity{EntityId: "entity1"})
	es1.add(searchName, 0.1)
	es2 := newEntitySim(&api.Entity{EntityId: "entity2"})
	es2.add(searchName, 0.3)
	es3 := newEntitySim(&api.Entity{EntityId: "entity3"})
	es3.add(searchName, 0.2)
	es4 := newEntitySim(&api.Entity{EntityId: "entity4"})
	es4.add(searchName, 0.4)

	ess := &entitySims{}
	heap.Push(ess, es1)
	heap.Push(ess, es2)
	heap.Push(ess, es3)
	heap.Push(ess, es4)

	// pop order should be ascending
	assert.Equal(t, es1.e.EntityId, ess.Peak().e.EntityId)
	assert.Equal(t, es1.e.EntityId, heap.Pop(ess).(*entitySim).e.EntityId)
	assert.Equal(t, es3.e.EntityId, heap.Pop(ess).(*entitySim).e.EntityId)
	assert.Equal(t, es2.e.EntityId, heap.Pop(ess).(*entitySim).e.EntityId)
	assert.Equal(t, es4.e.EntityId, heap.Pop(ess).(*entitySim).e.EntityId)
}

func TestEntitySim(t *testing.T) {
	es := newEntitySim(&api.Entity{})
	es.add("Search1", 0.2)
	es.add("Search2", 0.3)
	assert.Equal(t, []string{"Search1", "Search2"}, es.searches)
	assert.Equal(t, []float64{0.2, 0.3}, es.similarities)
	assert.Equal(t, math.Sqrt(0.2*0.2+0.3*0.3), es.similarity())
}

type fixedOfficeRows struct {
	ess    entitySims
	cursor int
}

func (fr *fixedOfficeRows) Scan(dest ...interface{}) error {
	e := fr.ess[fr.cursor].e
	f := e.TypeAttributes.(*api.Entity_Office).Office
	dest[0] = &e.EntityId
	dest[1] = &f.Name
	sim := fr.ess[fr.cursor].similarity()
	dest[2] = &sim
	fr.cursor++
	return nil
}

func (fr *fixedOfficeRows) Next() bool {
	return fr.cursor < len(fr.ess)
}

func (fr *fixedOfficeRows) Close() error {
	return nil
}
