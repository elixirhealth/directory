package memory

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

	"github.com/drausin/libri/libri/common/logging"
	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage"
	"github.com/elxirhealth/directory/pkg/server/storage/id"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestStorer_PutGetEntity_ok(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	idGen := id.NewNaiveLuhnGenerator(rng, id.DefaultLength)
	lg := logging.NewDevLogger(zapcore.DebugLevel)
	s := New(idGen, storage.NewDefaultParameters(), lg)
	assert.NotNil(t, s)

	cases := map[storage.EntityType]struct {
		original *api.Entity
		updated  *api.Entity
	}{
		storage.Patient: {
			original: api.NewPatient("", &api.Patient{
				LastName:  "Last Name 1",
				FirstName: "First Name 1",
				Birthdate: &api.Date{Year: 2006, Month: 1, Day: 2},
			}),
			updated: api.NewPatient("", &api.Patient{
				LastName:  "Last Name 2",
				FirstName: "First Name 1",
				Birthdate: &api.Date{Year: 2006, Month: 1, Day: 2},
			}),
		},

		storage.Office: {
			original: api.NewOffice("", &api.Office{
				Name: "Name 1",
			}),
			updated: api.NewOffice("", &api.Office{
				Name: "Name 2",
			}),
		},
	}
	assert.Equal(t, storage.NEntityTypes, len(cases))

	for et, c := range cases {
		assert.Equal(t, et, storage.GetEntityType(c.original), et.String())
		assert.NotEqual(t, c.original, c.updated)

		entityID, err := s.PutEntity(c.original)
		assert.Nil(t, err, et.String())
		assert.Equal(t, entityID, c.original.EntityId, et.String())
	}

	for et, c := range cases {
		entityID := c.original.EntityId
		gottenOriginal, err := s.GetEntity(entityID)
		assert.Nil(t, err, et.String())
		assert.Equal(t, c.original, gottenOriginal)

		c.updated.EntityId = c.original.EntityId
		entityID, err = s.PutEntity(c.updated)
		assert.Nil(t, err)
		assert.Equal(t, entityID, c.updated.EntityId)

		gottenUpdated, err := s.GetEntity(entityID)
		assert.Nil(t, err)
		assert.Equal(t, c.updated, gottenUpdated)
	}
}

func TestStorer_PutEntity_err(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	lg := zap.NewNop()
	okIDGen := id.NewNaiveLuhnGenerator(rng, id.DefaultLength)
	okID, err := okIDGen.Generate(storage.Patient.IDPrefix())
	assert.Nil(t, err)
	okEntity := api.NewTestPatient(0, false)

	cases := map[string]struct {
		s *storer
		e *api.Entity
	}{
		"bad entity ID": {
			s: &storer{
				idGen: okIDGen,
			},
			e: &api.Entity{EntityId: "bad ID"},
		},
		"bad entity": {
			s: &storer{
				idGen: okIDGen,
			},
			e: &api.Entity{},
		},
		"ID gen error": {
			s: &storer{
				idGen: &fixedIDGen{generateErr: errors.New("some Generate error")},
			},
			e: okEntity,
		},
	}

	for desc, c := range cases {
		entityID, err2 := c.s.PutEntity(c.e)
		assert.NotNil(t, err2, desc)
		assert.Empty(t, entityID, desc)
	}

	// two puts with same gen'd ID
	s := New(&fixedIDGen{generateID: okID}, storage.NewDefaultParameters(), lg)

	okEntity.EntityId = ""
	_, err = s.PutEntity(okEntity)
	assert.Nil(t, err)
	okEntity.EntityId = ""
	_, err = s.PutEntity(okEntity)
	assert.Equal(t, storage.ErrDupGenEntityID, err)
}

func TestStorer_GetEntity_err(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	idGen := id.NewNaiveLuhnGenerator(rng, id.DefaultLength)
	lg := zap.NewNop()
	s := New(idGen, storage.NewDefaultParameters(), lg)
	assert.NotNil(t, s)

	// bad ID
	e, err := s.GetEntity("bad ID")
	assert.NotNil(t, err)
	assert.Nil(t, e)

	// missing ID
	missingID, err := idGen.Generate(storage.Patient.IDPrefix())
	assert.Nil(t, err)
	e, err = s.GetEntity(missingID)
	assert.Equal(t, storage.ErrMissingEntity, err)
	assert.Nil(t, e)
}

func TestStorer_SearchEntity_ok(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	lg := logging.NewDevLogger(zapcore.DebugLevel)
	idGen := id.NewNaiveLuhnGenerator(rng, id.DefaultLength)
	s := New(idGen, storage.NewDefaultParameters(), lg)
	assert.NotNil(t, s)

	es := []*api.Entity{
		api.NewTestPatient(1, false),
		api.NewTestPatient(2, false),
		api.NewTestPatient(3, false),
		api.NewTestPatient(4, false),
		api.NewTestOffice(1, false),
		api.NewTestOffice(2, false),
		api.NewTestOffice(3, false),
		api.NewTestOffice(4, false),
	}
	entityIDs := make([]string, len(es))
	for i, e := range es {
		entityID, err2 := s.PutEntity(e)
		assert.Nil(t, err2)
		entityIDs[i] = entityID
	}

	limit := uint(3)

	query := "ice name 1" // query unanchored substring with diff case
	found, err := s.SearchEntity(query, limit)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(found))

	// check that first result is the office with the name that matches the query
	f, ok := found[0].TypeAttributes.(*api.Entity_Office)
	assert.True(t, ok)
	assert.True(t, strings.Contains(strings.ToUpper(f.Office.Name), strings.ToUpper(query)))

	// query 2nd patient's first 4 chars of entityID diff case
	query = strings.ToLower(entityIDs[1][:4])
	found, err = s.SearchEntity(query, limit)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(found))

	// check that first result is the patient with an entityID that matches the query
	_, ok = found[0].TypeAttributes.(*api.Entity_Patient)
	assert.True(t, ok)
	assert.True(t, strings.HasPrefix(found[0].EntityId, strings.ToUpper(query)))
}

func TestStorer_SearchEntity_err(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	idGen := id.NewNaiveLuhnGenerator(rng, id.DefaultLength)
	lg := zap.NewNop()
	params := storage.NewDefaultParameters()
	okStorer := New(idGen, params, lg)
	assert.NotNil(t, okStorer)

	okQuery := "some query"
	okLimit := uint(3)

	cases := map[string]struct {
		getStorer func() storage.Storer
		query     string
		limit     uint
		expected  error
	}{
		"query too short": {
			getStorer: func() storage.Storer { return okStorer },
			query:     "A",
			limit:     okLimit,
			expected:  api.ErrSearchQueryTooShort,
		},
		"query too long": {
			getStorer: func() storage.Storer { return okStorer },
			query:     strings.Repeat("A", 33),
			limit:     okLimit,
			expected:  api.ErrSearchQueryTooLong,
		},
		"limit too small": {
			getStorer: func() storage.Storer { return okStorer },
			query:     okQuery,
			limit:     0,
			expected:  api.ErrSearchLimitTooSmall,
		},
		"limit too large": {
			getStorer: func() storage.Storer { return okStorer },
			query:     okQuery,
			limit:     9,
			expected:  api.ErrSearchLimitTooLarge,
		},
	}

	for desc, c := range cases {
		s := c.getStorer()
		result, err := s.SearchEntity(c.query, c.limit)
		assert.Equal(t, c.expected, err, desc)
		assert.Nil(t, result, desc)
	}
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
