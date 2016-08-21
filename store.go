package kademlia

import (
	"crypto/sha1"
	"sync"
	"time"
)

// Store is the interface for implementing the storage mechanism for the
// DHT.
type Store interface {
	// Store should store a key/value pair on the network with the given
	// replication and expiration times.
	Store(key []byte, data []byte, replication time.Time, expiration time.Time, publisher bool) error

	// Retrieve searches the network a given key/value, or returns the
	// local key/value if it exists. If it is not found locally, or on
	// the network, the found return value will be false.
	Retrieve(key []byte) (data []byte, found bool)

	// Delete will delete a key/value pair from the Store
	Delete(key []byte)

	// Init initializes the Store
	Init()

	// GetAllKeysForReplication should return the keys of all data to be
	// replicated across the network. Typically all data should be
	// replicated every tReplicate seconds.
	GetAllKeysForReplication() [][]byte

	// ExpireKeys should expire all key/values due for expiration.
	ExpireKeys()

	// GetKey returns the key for data
	GetKey(data []byte) []byte
}

type MemoryStore struct {
	mutex        *sync.Mutex
	data         map[string][]byte
	replicateMap map[string]time.Time
	expireMap    map[string]time.Time
}

func (ms *MemoryStore) GetAllKeysForReplication() [][]byte {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	var keys [][]byte
	for k := range ms.data {
		if time.Now().After(ms.replicateMap[k]) {
			keys = append(keys, []byte(k))
		}
	}
	return keys
}

func (ms *MemoryStore) ExpireKeys() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	for k, v := range ms.expireMap {
		if time.Now().After(v) {
			delete(ms.replicateMap, k)
			delete(ms.expireMap, k)
			delete(ms.data, k)
		}
	}
}

func (ms *MemoryStore) Init() {
	ms.data = make(map[string][]byte)
	ms.mutex = &sync.Mutex{}
	ms.replicateMap = make(map[string]time.Time)
	ms.expireMap = make(map[string]time.Time)
}

func (ms *MemoryStore) GetKey(data []byte) []byte {
	sha := sha1.Sum(data)
	return sha[:]
}

func (ms *MemoryStore) Store(key []byte, data []byte, replication time.Time, expiration time.Time, publisher bool) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.replicateMap[string(key)] = replication
	ms.expireMap[string(key)] = expiration
	ms.data[string(key)] = data
	return nil
}

func (ms *MemoryStore) Retrieve(key []byte) (data []byte, found bool) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	data, found = ms.data[string(key)]
	return data, found
}

func (ms *MemoryStore) Delete(key []byte) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	delete(ms.replicateMap, string(key))
	delete(ms.expireMap, string(key))
	delete(ms.data, string(key))
}
