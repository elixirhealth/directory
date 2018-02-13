package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDirectory_ok(t *testing.T) {
	config := NewDefaultConfig()
	c, err := newDirectory(config)
	assert.Nil(t, err)
	// TODO assert NotNil on other elements of server struct
	assert.Equal(t, config, c.config)
}

func TestNewDirectory_err(t *testing.T) {
	badConfigs := map[string]*Config{
	// TODO add bad config instances
	}
	for desc, badConfig := range badConfigs {
		c, err := newDirectory(badConfig)
		assert.NotNil(t, err, desc)
		assert.Nil(t, c)
	}
}

// TODO add TestDirectory_ENDPOINT_(ok|err) for each ENDPOINT
