package dht

// Store TODO
type Store interface {
	Store(key []byte, data []byte) error
	Retrieve(key []byte) ([]byte, bool)
	Delete(key []byte)
	Init()
}

// MemoryStore TODO
type MemoryStore struct {
	data map[string][]byte
}

// Init TODO
func (ms *MemoryStore) Init() {
	ms.data = make(map[string][]byte)
}

// Store TODO
func (ms *MemoryStore) Store(key []byte, data []byte) error {
	ms.data[string(key)] = data
	return nil
}

// Retrieve TODO
func (ms *MemoryStore) Retrieve(key []byte) ([]byte, bool) {
	data, exists := ms.data[string(key)]
	return data, exists
}

// Delete TODO
func (ms *MemoryStore) Delete(key []byte) {
	delete(ms.data, string(key))
}
