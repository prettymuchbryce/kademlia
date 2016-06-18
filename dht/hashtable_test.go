package dht

import (
	"math"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Create a new node and bootstrap it. All nodes in the network know of a
// single node closer to the original node. This continues until every k bucket
// is occupied.
func TestFindNodeAllBuckets(t *testing.T) {
	networking := newMockNetworking()
	ht := newHashTable(&MemoryStore{}, networking, &Options{
		Port: "3000",
		IP:   "127.0.0.1",
	})
	id := getIDWithValues(0)
	ht.Self = &NetworkNode{
		ID:   id,
		Port: 3000,
		IP:   net.ParseIP("127.0.0.1"),
	}

	bootstrapNode := newNode(&NetworkNode{})
	bootstrapNode.ID = getZerodIDWithNthByte(0, byte(math.Pow(2, 7)))

	ht.addNode(bootstrapNode)

	var k = 0
	var i = 6

	go func() {
		for {
			queries := <-networking.recv
			responses := []*message{}
			for _, v := range queries {
				r := &message{}
				n := newNode(&NetworkNode{})
				n.ID = v.Sender.ID
				r.Receiver = n.NetworkNode
				r.Sender = &NetworkNode{ID: getIDWithValues(1), IP: net.ParseIP("127.0.0.1"), Port: 3001}

				responseData := &responseDataFindNode{}
				responseData.Closest = []*NetworkNode{&NetworkNode{ID: getZerodIDWithNthByte(k, byte(math.Pow(2, float64(i))))}}
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
			networking.send <- responses
		}
	}()

	ht.iterate(iterateFindNode, id, nil)

	for _, v := range ht.RoutingTable {
		assert.Equal(t, 1, len(v))
	}
}
