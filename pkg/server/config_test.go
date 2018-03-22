package server

import (
	"testing"

	"github.com/elxirhealth/directory/pkg/server/storage"
	bstorage "github.com/elxirhealth/service-base/pkg/server/storage"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	c := NewDefaultConfig()
	assert.NotNil(t, c)
	assert.NotEmpty(t, c.Storage)
}

func TestConfig_WithStorage(t *testing.T) {
	c1, c2, c3 := &Config{}, &Config{}, &Config{}
	c1.WithDefaultStorage()
	assert.Equal(t, c1.Storage.Type, c2.WithStorage(nil).Storage.Type)
	assert.NotEqual(t,
		c1.Storage.Type,
		c3.WithStorage(
			&storage.Parameters{Type: bstorage.Unspecified},
		).Storage.Type,
	)
}

func TestConfig_WithDBUrl(t *testing.T) {
	c1 := &Config{}
	dbURL := "some DB URL"
	c1.WithDBUrl(dbURL)
	assert.Equal(t, dbURL, c1.DBUrl)
}
