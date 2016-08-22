package kademlia

import (
	"bytes"
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
		BootstrapNodes: []*NetworkNode{{
			ID:   getZerodIDWithNthByte(0, byte(math.Pow(2, 7))),
			Port: 3001,
			IP:   net.ParseIP("0.0.0.0"),
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	go func() {
		dht.Listen()
	}()

	var k = 0
	var i = 6

	go func() {
		for {
			query := <-networking.recv
			if query == nil {
				return
			}

			res := mockFindNodeResponse(query, getZerodIDWithNthByte(k, byte(math.Pow(2, float64(i)))))

			i--
			if i < 0 {
				i = 7
				k++
			}
			if k > 19 {
				k = 19
			}

			networking.send <- res
		}
	}()

	dht.Bootstrap()

	for _, v := range dht.ht.RoutingTable {
		assert.Equal(t, 1, len(v))
	}

	dht.Disconnect()
}

// Tests timing out of nodes in a bucket. DHT bootstraps networks and learns
// about 20 subsequent nodes in the same bucket. Upon attempting to add the 21st
// node to the now full bucket, we should receive a ping to the very first node
// added in order to determine if it is still alive.
func TestAddNodeTimeout(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)
	done := make(chan (int))
	pinged := make(chan (int))

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
		BootstrapNodes: []*NetworkNode{{
			ID:   getZerodIDWithNthByte(1, byte(255)),
			Port: 3001,
			IP:   net.ParseIP("0.0.0.0"),
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	go func() {
		dht.Listen()
	}()

	var nodesAdded = 1
	var firstNode []byte
	var lastNode []byte

	go func() {
		for {
			query := <-networking.recv
			if query == nil {
				return
			}
			switch query.Type {
			case messageTypeFindNode:
				id := getIDWithValues(0)
				if nodesAdded > k+1 {
					close(done)
					return
				}

				if nodesAdded == 1 {
					firstNode = id
				}

				if nodesAdded == k {
					lastNode = id
				}

				id[1] = byte(255 - nodesAdded)
				nodesAdded++

				res := mockFindNodeResponse(query, id)
				networking.send <- res
			case messageTypePing:
				assert.Equal(t, messageTypePing, query.Type)
				assert.Equal(t, getZerodIDWithNthByte(1, byte(255)), query.Receiver.ID)
				close(pinged)
			}
		}
	}()

	dht.Bootstrap()

	// ensure the first node in the table is the second node contacted, and the
	// last is the last node contacted
	assert.Equal(t, 0, bytes.Compare(dht.ht.RoutingTable[b-9][0].ID, firstNode))
	assert.Equal(t, 0, bytes.Compare(dht.ht.RoutingTable[b-9][19].ID, lastNode))

	<-done
	<-pinged

	dht.Disconnect()
}

func TestGetRandomIDFromBucket(t *testing.T) {
	id := getIDWithValues(0)
	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
	})

	dht.CreateSocket()

	go func() {
		dht.Listen()
	}()

	// Bytes should be equal up to the bucket index that the random ID was
	// generated for, and random afterwards
	for i := 0; i < b/8; i++ {
		r := dht.ht.getRandomIDFromBucket(i * 8)
		for j := 0; j < i; j++ {
			assert.Equal(t, byte(0), r[j])
		}
	}

	dht.Disconnect()
}
