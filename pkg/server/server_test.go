package server

import (
	"context"
	"testing"

	api "github.com/elixirhealth/directory/pkg/directoryapi"
	"github.com/elixirhealth/directory/pkg/server/storage"
	"github.com/elixirhealth/service-base/pkg/server"
	bstorage "github.com/elixirhealth/service-base/pkg/server/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var okEntity = api.NewTestPatient(0, false)

func TestNewDirectory_ok(t *testing.T) {
	config := NewDefaultConfig().WithDBUrl("some DB URL")
	c, err := newDirectory(config)
	assert.Nil(t, err)
	assert.NotEmpty(t, c.storer)
	assert.Equal(t, config, c.config)
}

func TestNewDirectory_err(t *testing.T) {
	badConfigs := map[string]*Config{
		"empty DBUrl": NewDefaultConfig().
			WithDBUrl("").
			WithStorage(&storage.Parameters{Type: bstorage.Postgres}),
	}
	for desc, badConfig := range badConfigs {
		c, err := newDirectory(badConfig)
		assert.NotNil(t, err, desc)
		assert.Nil(t, c)
	}
}

func TestDirectory_PutEntity_ok(t *testing.T) {
	d := &Directory{
		BaseServer: server.NewBaseServer(server.NewDefaultBaseConfig()),
		storer: &fixedStorer{
			putEntityID: "some entity ID",
		},
	}
	rq := &api.PutEntityRequest{
		Entity: okEntity,
	}

	rp, err := d.PutEntity(context.Background(), rq)
	assert.Nil(t, err)
	assert.NotEmpty(t, rp.EntityId)
}

func TestDirectory_PutEntity_err(t *testing.T) {
	baseServer := server.NewBaseServer(server.NewDefaultBaseConfig())
	cases := map[string]struct {
		d  *Directory
		rq *api.PutEntityRequest
	}{
		"invalid request": {
			d: &Directory{
				BaseServer: baseServer,
			},
			rq: &api.PutEntityRequest{},
		},
		"storer Put error": {
			d: &Directory{
				BaseServer: baseServer,
				storer: &fixedStorer{
					putErr: errors.New("some Put error"),
				},
			},
			rq: &api.PutEntityRequest{
				Entity: okEntity,
			},
		},
	}

	for desc, c := range cases {
		rp, err := c.d.PutEntity(context.Background(), c.rq)
		assert.NotNil(t, err, desc)
		assert.Nil(t, rp, desc)
	}
}

func TestDirectory_GetEntity_ok(t *testing.T) {
	d := &Directory{
		BaseServer: server.NewBaseServer(server.NewDefaultBaseConfig()),
		storer: &fixedStorer{
			getEntity: okEntity,
		},
	}
	rq := &api.GetEntityRequest{
		EntityId: "some entity ID",
	}

	rp, err := d.GetEntity(context.Background(), rq)
	assert.Nil(t, err)
	assert.Equal(t, okEntity, rp.Entity)
}

func TestDirectory_GetEntity_err(t *testing.T) {
	baseServer := server.NewBaseServer(server.NewDefaultBaseConfig())
	cases := map[string]struct {
		d  *Directory
		rq *api.GetEntityRequest
	}{
		"invalid request": {
			d: &Directory{
				BaseServer: baseServer,
			},
			rq: &api.GetEntityRequest{},
		},
		"storer Get error": {
			d: &Directory{
				BaseServer: baseServer,
				storer: &fixedStorer{
					getErr: errors.New("some Get error"),
				},
			},
			rq: &api.GetEntityRequest{
				EntityId: "some entity ID",
			},
		},
	}

	for desc, c := range cases {
		rp, err := c.d.GetEntity(context.Background(), c.rq)
		assert.NotNil(t, err, desc)
		assert.Nil(t, rp, desc)
	}
}

func TestDirectory_SearchEntity_ok(t *testing.T) {
	d := &Directory{
		BaseServer: server.NewBaseServer(server.NewDefaultBaseConfig()),
		storer: &fixedStorer{
			searchEntities: []*api.Entity{
				api.NewTestPatient(0, true),
				api.NewTestPatient(1, true),
			},
		},
	}
	rq := &api.SearchEntityRequest{
		Query: "some query",
		Limit: 8,
	}

	rp, err := d.SearchEntity(context.Background(), rq)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rp.Entities))
}

func TestDirectory_SearchEntity_err(t *testing.T) {
	baseServer := server.NewBaseServer(server.NewDefaultBaseConfig())
	cases := map[string]struct {
		d  *Directory
		rq *api.SearchEntityRequest
	}{
		"invalid request": {
			d: &Directory{
				BaseServer: baseServer,
			},
			rq: &api.SearchEntityRequest{},
		},
		"storer Search error": {
			d: &Directory{
				BaseServer: baseServer,
				storer: &fixedStorer{
					searchErr: errors.New("some Search error"),
				},
			},
			rq: &api.SearchEntityRequest{
				Query: "some query",
			},
		},
	}

	for desc, c := range cases {
		rp, err := c.d.SearchEntity(context.Background(), c.rq)
		assert.NotNil(t, err, desc)
		assert.Nil(t, rp, desc)
	}
}

type fixedStorer struct {
	putEntityID    string
	putErr         error
	getEntity      *api.Entity
	getErr         error
	searchEntities []*api.Entity
	searchErr      error
	closeErr       error
}

func (f *fixedStorer) PutEntity(e *api.Entity) (string, error) {
	return f.putEntityID, f.putErr
}

func (f *fixedStorer) GetEntity(entityID string) (*api.Entity, error) {
	return f.getEntity, f.getErr
}

func (f *fixedStorer) SearchEntity(query string, limit uint) ([]*api.Entity, error) {
	return f.searchEntities, f.searchErr
}

func (f *fixedStorer) Close() error {
	return f.closeErr
}
