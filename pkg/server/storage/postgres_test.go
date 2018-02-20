package storage

import (
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
