package logging

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateLogger(t *testing.T) {
	level := "info"
	log := NewLogrus(level, os.Stdout)

	assert.Equal(t, log.level, level)
}

func TestGetLogger(t *testing.T) {
	level := "info"
	log := NewLogrus(level, os.Stdout)
	logger := log.Get("Testing")
	assert.Equal(t, logger.Logger.Out, os.Stdout)
}
