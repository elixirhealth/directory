package storage

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

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

func TestNewPostgres_err(t *testing.T) {
	rng := rand.New(rand.NewSource(0))
	idGen := NewNaiveIDGenerator(rng, DefaultIDLength)
	params := NewDefaultParameters()
	cases := map[string]struct {
		dbURL  string
		idGen  ChecksumIDGenerator
		params *Parameters
	}{
		"empty DB URL": {
			idGen:  idGen,
			params: params,
		},
		"wrong storage type": {
			dbURL: "some DB URL",
			idGen: idGen,
			params: &Parameters{
				Type: Unspecified,
			},
		},
	}

	for desc, c := range cases {
		s, err := NewPostgres(c.dbURL, c.idGen, c.params)
		assert.NotNil(t, err, desc)
		assert.Nil(t, s, desc)
	}
}

func TestPostgresStorer_PutGetEntity(t *testing.T) {
	dbURL, tearDown := setUpPostgresTest(t)
	defer func() {
		err := tearDown()
		assert.Nil(t, err)
	}()

	rng := rand.New(rand.NewSource(0))
	idGen := NewNaiveIDGenerator(rng, DefaultIDLength)
	s, err := NewPostgres(dbURL, idGen, NewDefaultParameters())
	assert.Nil(t, err)
	assert.NotNil(t, s)

	cases := map[entityType]struct {
		original *api.Entity
		updated  *api.Entity
	}{
		patient: {
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

		office: {
			original: api.NewOffice("", &api.Office{
				Name: "Name 1",
			}),
			updated: api.NewOffice("", &api.Office{
				Name: "Name 2",
			}),
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
	idGen := NewNaiveIDGenerator(rng, DefaultIDLength)
	s, err := NewPostgres(dbURL, idGen, NewDefaultParameters())
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
	okIDGen := NewNaiveIDGenerator(rng, DefaultIDLength)
	okID, err := okIDGen.Generate(patient.idPrefix())
	assert.Nil(t, err)
	okEntity := api.NewTestPatient(0, false)

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
		entityID, err2 := c.s.PutEntity(c.e)
		assert.NotNil(t, err2, desc)
		assert.Empty(t, entityID, desc)
	}

	// two puts with same gen'd ID
	s, err := NewPostgres(dbURL, &fixedIDGen{generateID: okID}, NewDefaultParameters())
	assert.Nil(t, err)
	okEntity.EntityId = ""
	_, err = s.PutEntity(okEntity)
	assert.Nil(t, err)
	okEntity.EntityId = ""
	_, err = s.PutEntity(okEntity)
	assert.Equal(t, ErrDupGenEntityID, err)
}

func TestPostgresStorer_SearchEntity(t *testing.T) {
	dbURL, tearDown := setUpPostgresTest(t)
	defer func() {
		err := tearDown()
		assert.Nil(t, err)
	}()

	rng := rand.New(rand.NewSource(0))
	idGen := NewNaiveIDGenerator(rng, DefaultIDLength)
	s, err := NewPostgres(dbURL, idGen, NewDefaultParameters())
	assert.Nil(t, err)
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
		entityID, err := s.PutEntity(e)
		entityIDs[i] = entityID
		assert.Nil(t, err)
	}

	limit := uint(3)

	query := "Office Name 1"
	found, err := s.SearchEntity(query, limit)
	assert.Nil(t, err)
	assert.Equal(t, limit, uint(len(found)))

	// check that first result is the office with the name that matches the query
	f, ok := found[0].TypeAttributes.(*api.Entity_Office)
	assert.True(t, ok)
	assert.Equal(t, query, f.Office.Name)

	// check that second and third results are also offices
	_, ok = found[1].TypeAttributes.(*api.Entity_Office)
	assert.True(t, ok)
	_, ok = found[2].TypeAttributes.(*api.Entity_Office)
	assert.True(t, ok)

	query = entityIDs[1] // 2nd patient
	found, err = s.SearchEntity(query, limit)
	assert.Nil(t, err)
	assert.Equal(t, limit, uint(len(found)))

	// check that first result is the patient with an entityID that matches the query
	_, ok = found[0].TypeAttributes.(*api.Entity_Patient)
	assert.True(t, ok)
	assert.Equal(t, query, found[0].EntityId)
}

func TestPreparePatientScan(t *testing.T) {
	e1 := api.NewTestPatient(0, true)
	p1 := e1.TypeAttributes.(*api.Entity_Patient).Patient

	cols, dest, scan := prepPatientScan(0)
	assert.Equal(t, len(cols), len(dest))

	// simulate row.Scan(dest...)
	dest[0] = &e1.EntityId
	dest[1] = &p1.LastName
	dest[2] = &p1.FirstName
	dest[3] = &p1.MiddleName
	dest[4] = &p1.Suffix
	birthdateTime, err := time.Parse("2006-01-02", p1.Birthdate.ISO8601())
	assert.Nil(t, err)
	dest[5] = &birthdateTime

	e2 := scan()
	assert.Equal(t, e1, e2)
}

func TestPrepareOfficeScan(t *testing.T) {
	e1 := &api.Entity{
		EntityId: "some entity ID",
		TypeAttributes: &api.Entity_Office{
			Office: &api.Office{
				Name: "Name 1",
			},
		},
	}
	f1 := e1.TypeAttributes.(*api.Entity_Office).Office

	cols, dest, scan := prepOfficeScan(0)
	assert.Equal(t, len(cols), len(dest))

	// simulate row.Scan(dest...)
	dest[0] = &e1.EntityId
	dest[1] = &f1.Name

	e2 := scan()
	assert.Equal(t, e1, e2)
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
