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
	id := getIDWithValues(0)

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
		BootstrapNodes: []*NetworkNode{&NetworkNode{
			ID:   getZerodIDWithNthByte(0, byte(math.Pow(2, 7))),
			Port: 3001,
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	var k = 0
	var i = 6

	go func() {
		for {
			v := <-networking.recv
			r := &message{}
			n := newNode(&NetworkNode{})
			n.ID = v.Sender.ID
			r.Receiver = n.NetworkNode
			r.Sender = &NetworkNode{ID: getIDWithValues(1), IP: net.ParseIP("0.0.0.0"), Port: 3001}

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
			networking.send <- r
		}
	}()

	dht.Bootstrap()

	for _, v := range dht.ht.RoutingTable {
		assert.Equal(t, 1, len(v))
	}
}

func TestNodeTimeout(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
		BootstrapNodes: []*NetworkNode{&NetworkNode{
			ID:   getZerodIDWithNthByte(9, byte(255)),
			Port: 3001,
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	var i = 254

	go func() {
		for {
			v := <-networking.recv
			r := &message{}
			switch v.Type {
			case messageTypeQueryFindNode:
				n := newNode(&NetworkNode{})
				n.ID = v.Sender.ID
				r.Receiver = n.NetworkNode
				r.Sender = &NetworkNode{ID: getIDWithValues(1), IP: net.ParseIP("0.0.0.0"), Port: 3001}

				id := getZerodIDWithNthByte(9, byte(math.Pow(2, 7)))
				if i < 255-k {
					i = 255
				}
				id[9] = byte(i)
				i--

				responseData := &responseDataFindNode{}
				responseData.Closest = []*NetworkNode{&NetworkNode{ID: id}}

				r.Data = responseData
				networking.send <- r
			case messageTypeQueryPing:
				assert.Equal(t, messageTypeQueryPing, v.Type)
			}
		}
	}()

	dht.Bootstrap()
}
