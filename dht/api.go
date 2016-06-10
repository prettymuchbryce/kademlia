package dht

import "crypto/sha256"

// Public interface

// DHT TODO
type DHT struct {
	ht *hashTable
}

// Options TODO
type Options struct {
	ID []byte
}

// New TODO
func (dht *DHT) New(store Store, options *Options) *DHT {
	dht.ht = newHashTable(store, &realNetworking{})
	return dht
}

// Store TODO
func (dht *DHT) Store(data []byte) {
	key := sha256.Sum256(data)
	dht.ht.Store.Store(key[:], data)
	go dht.ht.iterate(iterateStore, key[:], data)
}

// Get TODO
func (dht *DHT) Get(id []byte) {

}

// Connect TODO
func (dht *DHT) Connect() {

}

// Disconnect TODO
func (dht *DHT) Disconnect() {

}
