package dht

import (
	"crypto/sha1"

	b58 "github.com/jbenet/go-base58"
)

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
func NewDHT(store Store, options *Options) (*DHT, error) {
	dht := &DHT{}
	dht.options = options
	ht, err := newHashTable(store, &realNetworking{}, options)
	if err != nil {
		return nil, err
	}
	dht.ht = ht
	return dht, nil
}

// Store TODO
func (dht *DHT) Store(data []byte) string {
	sha := sha1.New()
	key := sha.Sum(data)
	dht.ht.Store.Store(key, data)
	go dht.ht.iterate(iterateStore, key[:], data)
	str := b58.Encode(key)
	return str
}

// Get TODO
func (dht *DHT) Get(key string) ([]byte, bool, error) {
	keyBytes := b58.Decode(key)
	value, exists := dht.ht.Store.Retrieve(keyBytes)
	if !exists {
		var err error
		value, _, err = dht.ht.iterate(iterateFindValue, keyBytes, nil)
		if err != nil {
			return nil, false, err
		}
		if value != nil {
			exists = true
		}
	}

	return value, exists, nil
}

// Connect TODO
func (dht *DHT) Connect() error {
	ip := dht.options.IP
	port := dht.options.Port
	if ip == "" {
		ip = "127.0.0.1"
	}
	if port == "" {
		port = "3000"
	}
	err := dht.ht.Networking.createSocket(ip, port)
	if err != nil {
		return err
	}
	go dht.ht.listen()
	go dht.ht.Networking.listen()
	if len(dht.options.BootstrapNodes) > 0 {
		for _, bn := range dht.options.BootstrapNodes {
			node := newNode(bn)
			dht.ht.addNode(node)
		}
	}
	_, _, err = dht.ht.iterate(iterateFindNode, dht.ht.Self.ID, nil)
	return err
}

// Disconnect TODO
func (dht *DHT) Disconnect() error {
	return dht.ht.Networking.disconnect()
}
