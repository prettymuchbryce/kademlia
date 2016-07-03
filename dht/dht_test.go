package dht

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapTwoNodes(t *testing.T) {
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
			dht2.Bootstrap()
			dht2.Disconnect()
			dht1.Disconnect()
		}()
		err := dht2.Listen()
		assert.Equal(t, "closed", err.Error())
	}()

	err = dht1.Listen()
	assert.Equal(t, "closed", err.Error())

	assert.Equal(t, 1, getTotalNodes(dht1.ht.RoutingTable))
	assert.Equal(t, 1, getTotalNodes(dht2.ht.RoutingTable))
}

func TestReconnect(t *testing.T) {
	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3004",
	})

	err := dht1.CreateSocket()
	assert.NoError(t, err)

	go func() {
		dht1.Disconnect()
	}()

	err = dht1.Listen()
	assert.Equal(t, "closed", err.Error())

	err = dht1.CreateSocket()
	assert.NoError(t, err)

	go func() {
		time.Sleep(1 * time.Second)
		dht1.Disconnect()
	}()

	err = dht1.Listen()
	assert.Equal(t, "closed", err.Error())
}

// func TestStoreAndFindValue(t *testing.T) {
// 	id1, _ := newID()
// 	dht1, _ := NewDHT(getInMemoryStore(), &Options{
// 		ID:   id1,
// 		IP:   "127.0.0.1",
// 		Port: "3002",
// 	})
//
// 	err := dht1.Connect()
// 	assert.NoError(t, err)
//
// 	key := dht1.Store([]byte("Foo"))
//
// 	dht2, _ := NewDHT(getInMemoryStore(), &Options{
// 		BootstrapNodes: []*NetworkNode{
// 			&NetworkNode{
// 				ID:   id1,
// 				IP:   net.ParseIP("127.0.0.1"),
// 				Port: 3002,
// 			},
// 		},
// 		IP:   "127.0.0.1",
// 		Port: "3003",
// 	})
//
// 	err = dht2.Connect()
// 	assert.NoError(t, err)
//
// 	value, exists, err := dht2.Get(key)
// 	assert.NoError(t, err)
// 	assert.Equal(t, true, exists)
// 	assert.Equal(t, []byte("Foo"), value)
// }

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
