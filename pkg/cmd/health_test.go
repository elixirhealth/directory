package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetHealthChecker(t *testing.T) {
	directories := "localhost:1234 localhost:5678"
	viper.Set(directoriesFlag, directories)
	hc, err := getHealthChecker()
	assert.Nil(t, err)
	assert.NotNil(t, hc)

	directories = "1234"
	viper.Set(directoriesFlag, directories)
	hc, err = getHealthChecker()
	assert.NotNil(t, err)
	assert.Nil(t, hc)
}
