package kademlia

import (
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistanceMetric(t *testing.T) {
	n := newNode(&NetworkNode{})
	n.ID = getIDWithValues(0)
	assert.Equal(t, 20, len(n.ID))

	value := getDistance(n.ID, getIDWithValues(0))
	assert.Equal(t, 0, value.Cmp(new(big.Int).SetInt64(int64(0))))

	v := getIDWithValues(0)
	v[19] = byte(1)
	value = getDistance(n.ID, v)
	assert.Equal(t, big.NewInt(1), value)

	v = getIDWithValues(0)
	v[18] = byte(1)
	value = getDistance(n.ID, v)
	assert.Equal(t, big.NewInt(256), value)

	v = getIDWithValues(255)
	value = getDistance(n.ID, v)

	// (2^160)-1 = max possible distance
	maxDistance := new(big.Int).Exp(big.NewInt(2), big.NewInt(160), nil)
	maxDistance.Sub(maxDistance, big.NewInt(1))

	assert.Equal(t, maxDistance, value)
}

func TestHasBit(t *testing.T) {
	for i := 0; i < 8; i++ {
		assert.Equal(t, true, hasBit(byte(255), uint(i)))
	}

	assert.Equal(t, true, hasBit(byte(1), uint(7)))

	for i := 0; i < 8; i++ {
		assert.Equal(t, false, hasBit(byte(0), uint(i)))
	}
}

func TestShortList(t *testing.T) {
	nl := &shortList{}
	comparator := getIDWithValues(0)
	n1 := &NetworkNode{ID: getZerodIDWithNthByte(19, 1)}
	n2 := &NetworkNode{ID: getZerodIDWithNthByte(18, 1)}
	n3 := &NetworkNode{ID: getZerodIDWithNthByte(17, 1)}
	n4 := &NetworkNode{ID: getZerodIDWithNthByte(16, 1)}

	nl.Nodes = []*NetworkNode{n3, n2, n4, n1}
	nl.Comparator = comparator

	sort.Sort(nl)

	assert.Equal(t, n1, nl.Nodes[0])
	assert.Equal(t, n2, nl.Nodes[1])
	assert.Equal(t, n3, nl.Nodes[2])
	assert.Equal(t, n4, nl.Nodes[3])
}

func getZerodIDWithNthByte(n int, v byte) []byte {
	id := getIDWithValues(0)
	id[n] = v
	return id
}

func getIDWithValues(b byte) []byte {
	return []byte{b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b, b}
}
