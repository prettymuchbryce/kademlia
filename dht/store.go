package dht

import "sync"

// Store TODO
type Store interface {
	Store(key []byte, data []byte) error
	Retrieve(key []byte) ([]byte, bool)
	Delete(key []byte)
	Init()
}

// MemoryStore TODO
type MemoryStore struct {
	data  map[string][]byte
	mutex *sync.Mutex
}

// Init TODO
func (ms *MemoryStore) Init() {
	ms.data = make(map[string][]byte)
	ms.mutex = &sync.Mutex{}
}

// Store TODO
func (ms *MemoryStore) Store(key []byte, data []byte) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
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
