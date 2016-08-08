package main

import (
	"sync"
	"time"
)

// Store TODO
type Store interface {
	Store(key []byte, data []byte, replication time.Time, expiration time.Time, publisher bool) error
	Retrieve(key []byte) ([]byte, bool)
	Delete(key []byte)
	Init()
	GetAllKeysForReplication() [][]byte
	ExpireKeys()
}

// MemoryStore TODO
type MemoryStore struct {
	mutex        *sync.Mutex
	data         map[string][]byte
	replicateMap map[string]time.Time
	expireMap    map[string]time.Time
}

// GetAllKeysForRefresh TODO
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

// ExpireKeys TODO
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

// Init TODO
func (ms *MemoryStore) Init() {
	ms.data = make(map[string][]byte)
	ms.mutex = &sync.Mutex{}
	ms.replicateMap = make(map[string]time.Time)
	ms.expireMap = make(map[string]time.Time)
}

// Store TODO
func (ms *MemoryStore) Store(key []byte, data []byte, replication time.Time, expiration time.Time, publisher bool) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.replicateMap[string(key)] = replication
	ms.expireMap[string(key)] = expiration
	ms.data[string(key)] = data
	return nil
}

// Retrieve TODO
func (ms *MemoryStore) Retrieve(key []byte) ([]byte, bool) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	data, exists := ms.data[string(key)]
	return data, exists
}

// Delete TODO
func (ms *MemoryStore) Delete(key []byte) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	delete(ms.data, string(key))
}
