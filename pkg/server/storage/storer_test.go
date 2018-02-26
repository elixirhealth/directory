package storage

import (
	"container/heap"
	"math"
	"math/rand"
	"testing"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage/id"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestEntitySims(t *testing.T) {
	searchName := "Search1"
	es1 := NewEntitySim(&api.Entity{EntityId: "entity1"})
	es1.Add(searchName, 0.1)
	es2 := NewEntitySim(&api.Entity{EntityId: "entity2"})
	es2.Add(searchName, 0.3)
	es3 := NewEntitySim(&api.Entity{EntityId: "entity3"})
	es3.Add(searchName, 0.2)
	es4 := NewEntitySim(&api.Entity{EntityId: "entity4"})
	es4.Add(searchName, 0.4)

	ess := &EntitySims{}
	heap.Push(ess, es1)
	heap.Push(ess, es2)
	heap.Push(ess, es3)
	heap.Push(ess, es4)

	// pop order should be ascending
	assert.Equal(t, es1.E.EntityId, ess.Peak().E.EntityId)
	assert.Equal(t, es1.E.EntityId, heap.Pop(ess).(*EntitySim).E.EntityId)
	assert.Equal(t, es3.E.EntityId, heap.Pop(ess).(*EntitySim).E.EntityId)
	assert.Equal(t, es2.E.EntityId, heap.Pop(ess).(*EntitySim).E.EntityId)
	assert.Equal(t, es4.E.EntityId, heap.Pop(ess).(*EntitySim).E.EntityId)
}

func TestEntitySim(t *testing.T) {
	es := NewEntitySim(&api.Entity{})
	es.Add("Search1", 0.2)
	es.Add("Search2", 0.3)
	assert.Equal(t, searcherSimilarities{"Search1": 0.2, "Search2": 0.3}, es.Similarities)
	assert.Equal(t, float32(math.Sqrt(0.2*0.2+0.3*0.3)), es.Similarity())
}

func TestMaybeAddEntityID(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	okIDGen := id.NewNaiveLuhnGenerator(rng, id.DefaultLength)

	added, err := MaybeAddEntityID(api.NewTestPatient(0, false), okIDGen)
	assert.Nil(t, err)
	assert.True(t, added)

	added, err = MaybeAddEntityID(api.NewTestPatient(0, true), okIDGen)
	assert.Nil(t, err)
	assert.False(t, added)

	errIDGen := &fixedIDGen{generateErr: errors.New("some Generate error")}
	added, err = MaybeAddEntityID(api.NewTestPatient(0, false), errIDGen)
	assert.NotNil(t, err)
	assert.False(t, added)
}

type fixedIDGen struct {
	checkErr    error
	generateID  string
	generateErr error
}

func (f *fixedIDGen) Check(id string) error {
	return f.checkErr
}

func (f *fixedIDGen) Generate(prefix string) (string, error) {
	return f.generateID, f.generateErr
}
