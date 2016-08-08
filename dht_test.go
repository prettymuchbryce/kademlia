package main

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Creates two DHTs, bootstrap one using the other, ensure that they both know
// about each other afterwards.
func TestBootstrapTwoNodes(t *testing.T) {
	done := make(chan bool)

	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	dht2, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			&NetworkNode{
				ID:   id1,
				IP:   net.ParseIP("127.0.0.1"),
				Port: 3000,
			},
		},
		IP:   "127.0.0.1",
		Port: "3001",
	})

	err := dht1.CreateSocket()
	assert.NoError(t, err)

	err = dht2.CreateSocket()
	assert.NoError(t, err)

	assert.Equal(t, 0, getTotalNodes(dht1.ht.RoutingTable))
	assert.Equal(t, 0, getTotalNodes(dht2.ht.RoutingTable))

	go func() {
		go func() {
			err := dht2.Bootstrap()
			assert.NoError(t, err)

			time.Sleep(50 * time.Millisecond)

			err = dht2.Disconnect()
			assert.NoError(t, err)

			err = dht1.Disconnect()
			assert.NoError(t, err)
			done <- true
		}()
		err := dht2.Listen()
		assert.Equal(t, "closed", err.Error())
		done <- true
	}()

	err = dht1.Listen()
	assert.Equal(t, "closed", err.Error())

	assert.Equal(t, 1, getTotalNodes(dht1.ht.RoutingTable))
	assert.Equal(t, 1, getTotalNodes(dht2.ht.RoutingTable))

	<-done
	<-done
}

// Create two DHTs have them connect and bootstrap, then disconnect. Repeat
// 100 times to ensure that we can use the same IP and port without EADDRINUSE
// errors.
func TestReconnect(t *testing.T) {
	for i := 0; i < 100; i++ {
		done := make(chan bool)

		id1, _ := newID()
		dht1, _ := NewDHT(getInMemoryStore(), &Options{
			ID:   id1,
			IP:   "127.0.0.1",
			Port: "3000",
		})

		dht2, _ := NewDHT(getInMemoryStore(), &Options{
			BootstrapNodes: []*NetworkNode{
				&NetworkNode{
					ID:   id1,
					IP:   net.ParseIP("127.0.0.1"),
					Port: 3000,
				},
			},
			IP:   "127.0.0.1",
			Port: "3001",
		})

		err := dht1.CreateSocket()
		assert.NoError(t, err)

		err = dht2.CreateSocket()
		assert.NoError(t, err)

		assert.Equal(t, 0, getTotalNodes(dht1.ht.RoutingTable))

		go func() {
			go func() {
				err := dht2.Bootstrap()
				assert.NoError(t, err)

				err = dht2.Disconnect()
				assert.NoError(t, err)

				err = dht1.Disconnect()
				assert.NoError(t, err)

				done <- true
			}()
			err := dht2.Listen()
			assert.Equal(t, "closed", err.Error())
			done <- true

		}()

		err = dht1.Listen()
		assert.Equal(t, "closed", err.Error())

		assert.Equal(t, 1, getTotalNodes(dht1.ht.RoutingTable))
		assert.Equal(t, 1, getTotalNodes(dht2.ht.RoutingTable))

		<-done
		<-done
	}
}

// Create two DHTs and have them connect. Send a store message from one node
// to another. Ensure that the other node now has this data in its store.
func TestStoreAndFindValue(t *testing.T) {
	done := make(chan bool)

	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	dht2, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			&NetworkNode{
				ID:   id1,
				IP:   net.ParseIP("127.0.0.1"),
				Port: 3000,
			},
		},
		IP:   "127.0.0.1",
		Port: "3001",
	})

	err := dht1.CreateSocket()
	assert.NoError(t, err)

	err = dht2.CreateSocket()
	assert.NoError(t, err)

	go func() {
		err := dht1.Listen()
		assert.Equal(t, "closed", err.Error())
		done <- true
	}()

	go func() {
		err := dht2.Listen()
		assert.Equal(t, "closed", err.Error())
		done <- true
	}()

	time.Sleep(1 * time.Second)

	dht2.Bootstrap()

	key, err := dht1.Store([]byte("Foo"))
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	value, exists, err := dht2.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, true, exists)
	assert.Equal(t, []byte("Foo"), value)

	err = dht1.Disconnect()
	assert.NoError(t, err)

	err = dht2.Disconnect()
	assert.NoError(t, err)

	<-done
	<-done
}

// Tests sending a message which results in an error when attempting to
// send over uTP
func TestNetworkingSendError(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)
	done := make(chan (int))

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
		BootstrapNodes: []*NetworkNode{&NetworkNode{
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

	go func() {
		v := <-networking.recv
		assert.Nil(t, v)
		close(done)
	}()

	networking.failNextSendMessage()

	dht.Bootstrap()

	dht.Disconnect()

	<-done
}

// Tests sending a message which results in a successful send, but the node
// never responds
func TestNodeResponseSendError(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)
	done := make(chan (int))

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id,
		Port: "3000",
		IP:   "0.0.0.0",
		BootstrapNodes: []*NetworkNode{&NetworkNode{
			ID:   getZerodIDWithNthByte(1, byte(255)),
			Port: 3001,
			IP:   net.ParseIP("0.0.0.0"),
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	queries := 0

	go func() {
		dht.Listen()
	}()

	go func() {
		for {
			query := <-networking.recv
			if query == nil {
				return
			}
			if queries == 1 {
				// Don't respond
				close(done)
			} else {
				queries++
				res := mockFindNodeResponse(query, getZerodIDWithNthByte(2, byte(255)))
				networking.send <- res
			}
		}
	}()

	dht.Bootstrap()

	assert.Equal(t, 1, dht.ht.totalNodes())

	dht.Disconnect()

	<-done
}

func TestBucketRefresh(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)
	done := make(chan (int))
	refresh := make(chan (int))

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:       id,
		Port:     "3000",
		IP:       "0.0.0.0",
		TRefresh: time.Second * 2,
		BootstrapNodes: []*NetworkNode{&NetworkNode{
			ID:   getZerodIDWithNthByte(1, byte(255)),
			Port: 3001,
			IP:   net.ParseIP("0.0.0.0"),
		},
		},
	})

	dht.networking = networking
	dht.CreateSocket()

	queries := 0

	go func() {
		dht.Listen()
	}()

	go func() {
		for {
			query := <-networking.recv
			if query == nil {
				close(done)
				return
			}
			queries++

			res := mockFindNodeResponseEmpty(query)
			networking.send <- res

			if queries == 2 {
				close(refresh)
			}
		}
	}()

	dht.Bootstrap()

	assert.Equal(t, 1, dht.ht.totalNodes())

	<-refresh

	dht.Disconnect()

	<-done
}

func getInMemoryStore() *MemoryStore {
	memStore := &MemoryStore{}
	memStore.Init()
	return memStore
}

func getTotalNodes(n [][]*node) int {
	j := 0
	for i := 0; i < len(n); i++ {
		j += len(n[i])
	}
	return j
}
