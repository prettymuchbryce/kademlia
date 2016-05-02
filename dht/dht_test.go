package dht

import (
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistanceMetric(t *testing.T) {
	n := &Node{}
	n.ID = getIDWithValues(0)
	assert.Equal(t, 20, len(n.ID))

	value := getDistance(n.ID, getIDWithValues(0))
	assert.Equal(t, 0, value.Cmp(new(big.Int).SetInt64(int64(0))))

	v := getIDWithValues(0)
	v[19] = byte(1)
	value = getDistance(n.ID, v)
	assert.Equal(t, 1, value)

	v = getIDWithValues(0)
	v[18] = byte(1)
	value = getDistance(n.ID, v)
	assert.Equal(t, 256, value)

	v = getIDWithValues(255)
	value = getDistance(n.ID, v)

	// (256^20)-1 = max possible distance
	maxDistance := new(big.Int).Exp(big.NewInt(256), big.NewInt(20), nil)
	maxDistance.Sub(maxDistance, big.NewInt(1))

	assert.Equal(t, maxDistance, value)
}

func TestNodeSort(t *testing.T) {
	nl := &NodeList{}
	comparator := getIDWithValues(0)
	n1 := &Node{ID: getZerodIDWithNthByte(19, 1)}
	n2 := &Node{ID: getZerodIDWithNthByte(18, 1)}
	n3 := &Node{ID: getZerodIDWithNthByte(17, 1)}
	n4 := &Node{ID: getZerodIDWithNthByte(16, 1)}

	nl.nodes = []*Node{n3, n2, n4, n1}
	nl.comparator = comparator

	sort.Sort(nl)

	assert.Equal(t, n1, nl.nodes[0])
	assert.Equal(t, n2, nl.nodes[1])
	assert.Equal(t, n3, nl.nodes[2])
	assert.Equal(t, n4, nl.nodes[3])
}

func getZerodIDWithNthByte(n int, v byte) []byte {
	id := getIDWithValues(0)
	id[n] = v
	return id
}

func getIDWithValues(b byte) []byte {
	return []byte{b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b}
}
