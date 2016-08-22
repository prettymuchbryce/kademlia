package kademlia

import (
	"bytes"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Creates twenty DHTs and bootstraps each with the previous
// at the end all should know about each other
func TestBootstrapTwentyNodes(t *testing.T) {
	done := make(chan bool)
	port := 3000
	dhts := []*DHT{}
	for i := 0; i < 20; i++ {
		id, _ := newID()
		dht, _ := NewDHT(getInMemoryStore(), &Options{
			ID:   id,
			IP:   "127.0.0.1",
			Port: strconv.Itoa(port),
			BootstrapNodes: []*NetworkNode{
				NewNetworkNode("127.0.0.1", strconv.Itoa(port-1)),
			},
		})
		port++
		dhts = append(dhts, dht)
		err := dht.CreateSocket()
		assert.NoError(t, err)
	}

	for _, dht := range dhts {
		assert.Equal(t, 0, dht.NumNodes())
		go func(dht *DHT) {
			err := dht.Listen()
			assert.Equal(t, "closed", err.Error())
			done <- true
		}(dht)
		go func(dht *DHT) {
			err := dht.Bootstrap()
			assert.NoError(t, err)
		}(dht)
		time.Sleep(time.Millisecond * 200)
	}

	time.Sleep(time.Millisecond * 2000)

	for _, dht := range dhts {
		assert.Equal(t, 19, dht.NumNodes())
		err := dht.Disconnect()
		assert.NoError(t, err)
		<-done
	}
}

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
			{
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

	assert.Equal(t, 0, dht1.NumNodes())
	assert.Equal(t, 0, dht2.NumNodes())

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

	assert.Equal(t, 1, dht1.NumNodes())
	assert.Equal(t, 1, dht2.NumNodes())
	<-done
	<-done
}

// Creates three DHTs, bootstrap B using A, bootstrap C using B. A should know
// about both B and C
func TestBootstrapThreeNodes(t *testing.T) {
	done := make(chan bool)

	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	id2, _ := newID()
	dht2, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			{
				ID:   id1,
				IP:   net.ParseIP("127.0.0.1"),
				Port: 3000,
			},
		},
		IP:   "127.0.0.1",
		Port: "3001",
		ID:   id2,
	})

	dht3, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			{
				ID:   id2,
				IP:   net.ParseIP("127.0.0.1"),
				Port: 3001,
			},
		},
		IP:   "127.0.0.1",
		Port: "3002",
	})

	err := dht1.CreateSocket()
	assert.NoError(t, err)

	err = dht2.CreateSocket()
	assert.NoError(t, err)

	err = dht3.CreateSocket()
	assert.NoError(t, err)

	assert.Equal(t, 0, dht1.NumNodes())
	assert.Equal(t, 0, dht2.NumNodes())
	assert.Equal(t, 0, dht3.NumNodes())

	go func(dht1 *DHT, dht2 *DHT, dht3 *DHT) {
		go func(dht1 *DHT, dht2 *DHT, dht3 *DHT) {
			err := dht2.Bootstrap()
			assert.NoError(t, err)

			go func(dht1 *DHT, dht2 *DHT, dht3 *DHT) {
				err := dht3.Bootstrap()
				assert.NoError(t, err)

				time.Sleep(500 * time.Millisecond)

				err = dht1.Disconnect()
				assert.NoError(t, err)

				time.Sleep(100 * time.Millisecond)

				err = dht2.Disconnect()
				assert.NoError(t, err)

				err = dht3.Disconnect()
				assert.NoError(t, err)
				done <- true
			}(dht1, dht2, dht3)

			err = dht3.Listen()
			assert.Equal(t, "closed", err.Error())
			done <- true
		}(dht1, dht2, dht3)
		err := dht2.Listen()
		assert.Equal(t, "closed", err.Error())
		done <- true
	}(dht1, dht2, dht3)

	err = dht1.Listen()
	assert.Equal(t, "closed", err.Error())

	assert.Equal(t, 2, dht1.NumNodes())
	assert.Equal(t, 2, dht2.NumNodes())
	assert.Equal(t, 2, dht3.NumNodes())

	<-done
	<-done
	<-done
}

// Creates two DHTs and bootstraps using only IP:Port. Connecting node should
// ping the first node to find its ID
func TestBootstrapNoID(t *testing.T) {
	done := make(chan bool)

	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	dht2, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			{
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

	assert.Equal(t, 0, dht1.NumNodes())
	assert.Equal(t, 0, dht2.NumNodes())

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

	assert.Equal(t, 1, dht1.NumNodes())
	assert.Equal(t, 1, dht2.NumNodes())

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
				{
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

		assert.Equal(t, 0, dht1.NumNodes())

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

		assert.Equal(t, 1, dht1.NumNodes())
		assert.Equal(t, 1, dht2.NumNodes())

		<-done
		<-done
	}
}

// Create two DHTs and have them connect. Send a store message with 100mb
// payload from one node to another. Ensure that the other node now has
// this data in its store.
func TestStoreAndFindLargeValue(t *testing.T) {
	done := make(chan bool)

	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	dht2, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			{
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

	payload := [1000000]byte{}

	key, err := dht1.Store(payload[:])
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	value, exists, err := dht2.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, true, exists)
	assert.Equal(t, 0, bytes.Compare(payload[:], value))

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
		BootstrapNodes: []*NetworkNode{{
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

// Tests a bucket refresh by setting a very low TRefresh value, adding a single
// node to a bucket, and waiting for the refresh message for the bucket
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
		BootstrapNodes: []*NetworkNode{{
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

// Tets store replication by setting the TReplicate time to a very small value.
// Stores some data, and then expects another store message in TReplicate time
func TestStoreReplication(t *testing.T) {
	networking := newMockNetworking()
	id := getIDWithValues(0)
	done := make(chan (int))
	replicate := make(chan (int))

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:         id,
		Port:       "3000",
		IP:         "0.0.0.0",
		TReplicate: time.Second * 2,
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

	stores := 0

	go func() {
		for {
			query := <-networking.recv
			if query == nil {
				close(done)
				return
			}

			switch query.Type {
			case messageTypeFindNode:
				res := mockFindNodeResponseEmpty(query)
				networking.send <- res
			case messageTypeStore:
				stores++
				d := query.Data.(*queryDataStore)
				assert.Equal(t, []byte("foo"), d.Data)
				if stores == 2 {
					close(replicate)
				}
			}
		}
	}()

	dht.Bootstrap()

	dht.Store([]byte("foo"))

	<-replicate

	dht.Disconnect()

	<-done
}

// Test Expiration by setting TExpire to a very low value. Store a value,
// and then wait longer than TExpire. The value should no longer exist in
// the store.
func TestStoreExpiration(t *testing.T) {
	id := getIDWithValues(0)

	dht, _ := NewDHT(getInMemoryStore(), &Options{
		ID:      id,
		Port:    "3000",
		IP:      "0.0.0.0",
		TExpire: time.Second,
	})

	dht.CreateSocket()

	go func() {
		dht.Listen()
	}()

	k, _ := dht.Store([]byte("foo"))

	v, exists, _ := dht.Get(k)
	assert.Equal(t, true, exists)

	assert.Equal(t, []byte("foo"), v)

	<-time.After(time.Second * 3)

	_, exists, _ = dht.Get(k)

	assert.Equal(t, false, exists)

	dht.Disconnect()
}

func getInMemoryStore() *MemoryStore {
	memStore := &MemoryStore{}
	return memStore
}
