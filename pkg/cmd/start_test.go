package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestGetDirectoryConfig(t *testing.T) {
	serverPort := uint(1234)
	metricsPort := uint(5678)
	profilerPort := uint(9012)
	logLevel := zapcore.DebugLevel.String()
	profile := true
	// TODO add other non-default config values

	viper.Set(serverPortFlag, serverPort)
	viper.Set(metricsPortFlag, metricsPort)
	viper.Set(profilerPortFlag, profilerPort)
	viper.Set(logLevelFlag, logLevel)
	viper.Set(profileFlag, profile)
	// TODO set other non-default config value

	c, err := getDirectoryConfig()
	assert.Nil(t, err)
	assert.Equal(t, serverPort, c.ServerPort)
	assert.Equal(t, metricsPort, c.MetricsPort)
	assert.Equal(t, profilerPort, c.ProfilerPort)
	assert.Equal(t, logLevel, c.LogLevel.String())
	assert.Equal(t, profile, c.Profile)
	// TODO assert equal other non-default config values

}
