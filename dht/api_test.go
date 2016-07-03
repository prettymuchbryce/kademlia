package dht

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapAPITwoNodes(t *testing.T) {
	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	err := dht1.Connect()
	assert.NoError(t, err)

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

	assert.Equal(t, 0, getTotalNodes(dht1.ht.RoutingTable))

	err = dht2.Connect()
	assert.NoError(t, err)

	assert.Equal(t, 1, getTotalNodes(dht1.ht.RoutingTable))

	err = dht1.Disconnect()
	assert.NoError(t, err)
	err = dht2.Disconnect()
	assert.NoError(t, err)
}

func TestDisconnect(t *testing.T) {
	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3004",
	})

	err := dht1.Connect()
	assert.NoError(t, err)
	//
	// dht2 := NewDHT(getInMemoryStore(), &Options{
	// 	BootstrapNodes: []*NetworkNode{
	// 		&NetworkNode{
	// 			ID:   id1,
	// 			IP:   net.ParseIP("127.0.0.1"),
	// 			Port: 3004,
	// 		},
	// 	},
	// 	IP:   "127.0.0.1",
	// 	Port: "3005",
	// })

	// err = dht2.Connect()
	// assert.NoError(t, err)

	// time.Sleep(2 * time.Second)

	err = dht1.Disconnect()
	assert.NoError(t, err)

	err = dht1.Connect()
	assert.NoError(t, err)
}

func TestStoreAndFindValue(t *testing.T) {
	id1, _ := newID()
	dht1, _ := NewDHT(getInMemoryStore(), &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3002",
	})

	err := dht1.Connect()
	assert.NoError(t, err)

	key := dht1.Store([]byte("Foo"))

	dht2, _ := NewDHT(getInMemoryStore(), &Options{
		BootstrapNodes: []*NetworkNode{
			&NetworkNode{
				ID:   id1,
				IP:   net.ParseIP("127.0.0.1"),
				Port: 3002,
			},
		},
		IP:   "127.0.0.1",
		Port: "3003",
	})

	err = dht2.Connect()
	assert.NoError(t, err)

	value, exists, err := dht2.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, true, exists)
	assert.Equal(t, []byte("Foo"), value)
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
