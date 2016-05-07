package dht

import (
	"math"
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistanceMetric(t *testing.T) {
	n := &node{}
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

	// (2^160)-1 = max possible distance
	maxDistance := new(big.Int).Exp(big.NewInt(2), big.NewInt(160), nil)
	maxDistance.Sub(maxDistance, big.NewInt(1))

	assert.Equal(t, maxDistance, value)
}

// A new node and bootstrap it. All nodes in the network know of a single node
// closer to the original node. This continues until every k bucket is occupied.
func TestFindNodeAllBuckets(t *testing.T) {
	ht := newHashTable()
	id := getIDWithValues(0)
	ht.ID = id

	net := newMockNetworking()
	bootstrapNode := &node{}
	bootstrapNode.ID = getZerodIDWithNthByte(0, byte(math.Pow(2, 7)))
	ht.Networking = net

	ht.addNode(bootstrapNode)

	var k = 0
	var i = 6

	go func() {
		for {
			queries := <-net.recv
			responses := []*response{}
			for _, v := range queries {
				r := &response{}
				n := &node{}
				n.ID = v.Node.ID
				r.Node = n

				responseData := &responseDataFindNode{}
				responseData.Closest = []*node{&node{ID: getZerodIDWithNthByte(k, byte(math.Pow(2, float64(i))))}}
				i--
				if i < 0 {
					i = 7
					k++
				}
				if k > 19 {
					k = 19
				}

				r.Data = responseData
				responses = append(responses, r)
			}
			net.send <- responses
		}
	}()

	ht.iterate(iterateFindNode, id, nil)

	for _, v := range ht.RoutingTable {
		assert.Equal(t, 1, len(v))
	}

}

func TestShortList(t *testing.T) {
	nl := &shortList{}
	comparator := getIDWithValues(0)
	n1 := &node{ID: getZerodIDWithNthByte(19, 1)}
	n2 := &node{ID: getZerodIDWithNthByte(18, 1)}
	n3 := &node{ID: getZerodIDWithNthByte(17, 1)}
	n4 := &node{ID: getZerodIDWithNthByte(16, 1)}

	nl.Nodes = []*node{n3, n2, n4, n1}
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
