package storage

import (
	"testing"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/stretchr/testify/assert"
)

func TestEntityType_string(t *testing.T) {
	for i := 0; i < nEntityTypes; i++ {
		et := entityType(i)
		assert.NotEmpty(t, et.string())
	}
}

func TestEntityType_idPrefix(t *testing.T) {
	for i := 0; i < nEntityTypes; i++ {
		et := entityType(i)
		assert.NotEmpty(t, et.idPrefix())
	}
}

func TestGetEntityType(t *testing.T) {
	cases := map[entityType]*api.Entity{
		patient: {TypeAttributes: &api.Entity_Patient{}},
		office:  {TypeAttributes: &api.Entity_Office{}},
	}
	assert.Equal(t, nEntityTypes, len(cases))
	for et, e := range cases {
		assert.Equal(t, et, getEntityType(e))
	}
}

func TestGetEntityTypeFromID(t *testing.T) {
	cases := map[entityType]string{
		patient: "PAAAAAAA",
		office:  "FAAAAAAA",
	}
	assert.Equal(t, nEntityTypes, len(cases))
	for et, id := range cases {
		assert.Equal(t, et, getEntityTypeFromID(id))
	}
}
