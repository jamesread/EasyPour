package buildinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion_DefaultsToDevWhenNotSetByLinker(t *testing.T) {
	assert.Equal(t, "dev", Version)
}
