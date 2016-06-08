package dht

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Create a new node and bootstrap it. All nodes in the network know of a
// single node closer to the original node. This continues until every k bucket
// is occupied.
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
			responses := []*message{}
			for _, v := range queries {
				r := &message{}
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
