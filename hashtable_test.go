package main

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
			IP:   net.ParseIP("0.0.0.0"),
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
			n.IP = v.Sender.IP
			n.Port = v.Sender.Port
			r.Receiver = n.NetworkNode
			r.Sender = &NetworkNode{ID: v.Receiver.ID, IP: net.ParseIP("0.0.0.0"), Port: 3001}
			r.Type = v.Type
			r.IsResponse = true
			responseData := &responseDataFindNode{}
			responseData.Closest = []*NetworkNode{&NetworkNode{IP: net.ParseIP("0.0.0.0"), Port: 3001, ID: getZerodIDWithNthByte(k, byte(math.Pow(2, float64(i))))}}
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

// Tests timing out of nodes in a bucket. DHT bootstraps networks and learns
// about 20 subsequent nodes in the same bucket. Upon attempting to add the 21st
// node to the now full bucket, we should recieve a ping to the very first node
// added in order to determine if it is still alive.
func TestNodeTimeout(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)
	done := make(chan (int))
	pinged := make(chan (int))

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
		BootstrapNodes: []*NetworkNode{&NetworkNode{
			ID:   getZerodIDWithNthByte(9, byte(255)),
			Port: 3001,
			IP:   net.ParseIP("0.0.0.0"),
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	var nodesAdded = 1

	go func() {
		for {
			v := <-networking.recv
			r := &message{}
			switch v.Type {
			case messageTypeFindNode:
				n := newNode(&NetworkNode{})
				n.ID = v.Sender.ID
				n.IP = v.Sender.IP
				n.Port = v.Sender.Port
				r.Receiver = n.NetworkNode
				r.Sender = &NetworkNode{ID: v.Receiver.ID, IP: net.ParseIP("0.0.0.0"), Port: 3001}
				r.Type = v.Type
				r.IsResponse = true

				id := getIDWithValues(0)
				if nodesAdded > k {
					close(done)
					return
				}
				id[9] = byte(255 - nodesAdded)
				nodesAdded++

				responseData := &responseDataFindNode{}
				responseData.Closest = []*NetworkNode{&NetworkNode{ID: id, IP: net.ParseIP("0.0.0.0"), Port: 3001}}

				r.Data = responseData
				networking.send <- r
			case messageTypePing:
				assert.Equal(t, messageTypePing, v.Type)
				assert.Equal(t, getZerodIDWithNthByte(9, byte(255)), v.Receiver.ID)
				close(pinged)
			}
		}
	}()

	dht.Bootstrap()

	<-done
	<-pinged
}
