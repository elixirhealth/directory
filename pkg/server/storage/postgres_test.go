package storage

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"testing"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/elxirhealth/directory/pkg/server/storage/migrations"
	"github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/mattes/migrate/source/go-bindata"
	"github.com/stretchr/testify/assert"
)

var (
	setUpPostgresTest func(t *testing.T) (dbURL string, tearDown func() error)
)

func TestMain(m *testing.M) {
	dbURL, cleanup, err := storage.StartTestPostgres()
	if err != nil {
		if err2 := cleanup(); err2 != nil {
			log.Fatal(err2.Error())
		}
		log.Fatal(err.Error())
	}
	setUpPostgresTest = func(t *testing.T) (string, func() error) {
		return dbURL, storage.SetUpTestPostgresDB(t, dbURL, bindata.Resource(
			migrations.AssetNames(),
			func(name string) ([]byte, error) { return migrations.Asset(name) },
		))
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := cleanup(); err != nil {
		log.Fatal(err.Error())
	}

	os.Exit(code)
}

func TestPostgresStorer_PutGetEntity(t *testing.T) {
	dbURL, tearDown := setUpPostgresTest(t)
	defer func() {
		err := tearDown()
		assert.Nil(t, err)
	}()

	rng := rand.New(rand.NewSource(0))
	idGen := NewNaiveIDGenerator(rng, 9)
	s, err := NewPostgresStorer(dbURL, idGen)
	assert.Nil(t, err)
	assert.NotNil(t, s)

	cases := map[entityType]struct {
		original *api.Entity
		updated  *api.Entity
	}{
		patient: {
			original: &api.Entity{
				TypeAttributes: &api.Entity_Patient{
					Patient: &api.Patient{
						LastName:   "Last Name 1",
						FirstName:  "First Name 1",
						MiddleName: "Middle Name 1",
						Birthdate:  &api.Date{Year: 2006, Month: 1, Day: 2},
					},
				},
			},
			updated: &api.Entity{
				TypeAttributes: &api.Entity_Patient{
					Patient: &api.Patient{
						LastName:   "Last Name 2",
						FirstName:  "First Name 1",
						MiddleName: "Middle Name 1",
						Birthdate:  &api.Date{Year: 2006, Month: 1, Day: 2},
					},
				},
			},
		},

		office: {
			original: &api.Entity{
				TypeAttributes: &api.Entity_Office{
					Office: &api.Office{
						Name: "Name 1",
					},
				},
			},
			updated: &api.Entity{
				TypeAttributes: &api.Entity_Office{
					Office: &api.Office{
						Name: "Name 2",
					},
				},
			},
		},
	}
	assert.Equal(t, nEntityTypes, len(cases))

	for et, c := range cases {
		assert.Equal(t, et, getEntityType(c.original), et.string())
		assert.NotEqual(t, c.original, c.updated)

		entityID, err := s.PutEntity(c.original)
		assert.Nil(t, err, et.string())
		assert.Equal(t, entityID, c.original.EntityId, et.string())

		gottenOriginal, err := s.GetEntity(entityID)
		assert.Nil(t, err, et.string())
		assert.Equal(t, c.original, gottenOriginal)

		c.updated.EntityId = entityID
		entityID, err = s.PutEntity(c.updated)
		assert.Nil(t, err)
		assert.Equal(t, entityID, c.updated.EntityId)

		gottenUpdated, err := s.GetEntity(entityID)
		assert.Nil(t, err)
		assert.Equal(t, c.updated, gottenUpdated)
	}
}

func TestPostgresStorer_GetEntity_err(t *testing.T) {
	dbURL, tearDown := setUpPostgresTest(t)
	defer func() {
		err := tearDown()
		assert.Nil(t, err)
	}()

	rng := rand.New(rand.NewSource(0))
	idGen := NewNaiveIDGenerator(rng, 9)
	s, err := NewPostgresStorer(dbURL, idGen)
	assert.Nil(t, err)
	assert.NotNil(t, s)

	// bad ID
	e, err := s.GetEntity("bad ID")
	assert.NotNil(t, err)
	assert.Nil(t, e)

	// missing ID
	missingID, err := idGen.Generate(patient.idPrefix())
	assert.Nil(t, err)
	e, err = s.GetEntity(missingID)
	assert.Equal(t, ErrMissingEntity, err)
	assert.Nil(t, e)
}

func TestPostgresStorer_PutEntity_err(t *testing.T) {
	dbURL, tearDown := setUpPostgresTest(t)
	defer func() {
		err := tearDown()
		assert.Nil(t, err)
	}()

	rng := rand.New(rand.NewSource(0))
	okIDGen := NewNaiveIDGenerator(rng, 9)
	okID, err := okIDGen.Generate(patient.idPrefix())
	assert.Nil(t, err)
	okEntity := &api.Entity{
		TypeAttributes: &api.Entity_Patient{
			Patient: &api.Patient{
				LastName:   "Last Name 1",
				FirstName:  "First Name 1",
				MiddleName: "Middle Name 1",
				Birthdate:  &api.Date{Year: 2006, Month: 1, Day: 2},
			},
		},
	}

	cases := map[string]struct {
		s *postgresStorer
		e *api.Entity
	}{
		"bad entity ID": {
			s: &postgresStorer{
				idGen: okIDGen,
			},
			e: &api.Entity{EntityId: "bad ID"},
		},
		"bad entity": {
			s: &postgresStorer{
				idGen: okIDGen,
			},
			e: &api.Entity{},
		},
		"ID gen error": {
			s: &postgresStorer{
				idGen: &fixedIDGen{generateErr: errors.New("some Generate error")},
			},
			e: okEntity,
		},
	}

	for desc, c := range cases {
		entityID, err := c.s.PutEntity(c.e)
		assert.NotNil(t, err, desc)
		assert.Empty(t, entityID, desc)
	}

	// two puts with same gen'd ID
	s, err := NewPostgresStorer(dbURL, &fixedIDGen{generateID: okID})
	okEntity.EntityId = ""
	_, err = s.PutEntity(okEntity)
	assert.Nil(t, err)
	okEntity.EntityId = ""
	_, err = s.PutEntity(okEntity)
	assert.Equal(t, ErrDupGenEntityID, err)
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
