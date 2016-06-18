package dht

import "crypto/sha256"

// Public interface

// DHT TODO
type DHT struct {
	ht      *hashTable
	options *Options
}

// Options TODO
type Options struct {
	ID             []byte
	IP             string
	Port           string
	BootstrapNodes []*NetworkNode
}

// NewDHT TODO
func NewDHT(store Store, options *Options) *DHT {
	dht := &DHT{}
	dht.options = options
	dht.ht = newHashTable(store, &realNetworking{}, options)
	return dht
}

// Store TODO
func (dht *DHT) Store(data []byte) {
	key := sha256.Sum256(data)
	dht.ht.Store.Store(key[:], data)
	go dht.ht.iterate(iterateStore, key[:], data)
}

// Get TODO
func (dht *DHT) Get(key []byte) ([]byte, bool) {
	value, exists := dht.ht.Store.Retrieve(key)
	if !exists {
		value, _ = dht.ht.iterate(iterateFindValue, key, nil)
		if value != nil {
			exists = true
		}
	}

	return value, exists
}

// Connect TODO
func (dht *DHT) Connect() {
	ip := dht.options.IP
	port := dht.options.Port
	if ip == "" {
		ip = "127.0.0.1"
	}
	if port == "" {
		port = "3000"
	}
	dht.ht.Networking.createSocket(ip, port)
	go dht.ht.listen()
	go dht.ht.Networking.listen()
	if len(dht.options.BootstrapNodes) > 0 {
		for _, bn := range dht.options.BootstrapNodes {
			node := newNode(&NetworkNode{})
			node.ID = bn.ID
			node.IP = bn.IP
			node.Port = bn.Port
			dht.ht.addNode(node)
			dht.ht.iterate(iterateFindNode, node.ID, nil)
		}
	}
}

// Disconnect TODO
func (dht *DHT) Disconnect() {

}
