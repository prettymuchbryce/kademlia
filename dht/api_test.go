package dht

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAPI(t *testing.T) {
	id1, _ := newID()
	dht1 := NewDHT(&MemoryStore{}, &Options{
		ID:   id1,
		IP:   "127.0.0.1",
		Port: "3000",
	})

	dht1.Connect()

	dht2 := NewDHT(&MemoryStore{}, &Options{
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

	go dht2.Connect()

	time.Sleep(time.Second * 2)

	assert.Equal(t, 1, getTotalNodes(dht1.ht.RoutingTable))
}

func getTotalNodes(n [][]*node) int {
	j := 0
	for i := 0; i < len(n); i++ {
		j += len(n[i])
	}
	return j
}
