package dht

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistanceMetric(t *testing.T) {
	n := &Node{}
	n.ID = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.Equal(t, 20, len(n.ID))
}
