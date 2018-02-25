package storage

import (
	"testing"

	api "github.com/elxirhealth/directory/pkg/directoryapi"
	"github.com/stretchr/testify/assert"
)

func TestEntityType_string(t *testing.T) {
	for i := 0; i < NEntityTypes; i++ {
		et := EntityType(i)
		assert.NotEmpty(t, et.String())
	}
}

func TestEntityType_idPrefix(t *testing.T) {
	for i := 0; i < NEntityTypes; i++ {
		et := EntityType(i)
		assert.NotEmpty(t, et.IDPrefix())
	}
}

func TestGetEntityType(t *testing.T) {
	cases := map[EntityType]*api.Entity{
		Patient: {TypeAttributes: &api.Entity_Patient{}},
		Office:  {TypeAttributes: &api.Entity_Office{}},
	}
	assert.Equal(t, NEntityTypes, len(cases))
	for et, e := range cases {
		assert.Equal(t, et, GetEntityType(e))
	}
}

func TestGetEntityTypeFromID(t *testing.T) {
	cases := map[EntityType]string{
		Patient: "PAAAAAAA",
		Office:  "FAAAAAAA",
	}
	assert.Equal(t, NEntityTypes, len(cases))
	for et, id := range cases {
		assert.Equal(t, et, GetEntityTypeFromID(id))
	}
}
