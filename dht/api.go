package dht

// Public interface

// DHT TODO
type DHT struct {
	queue *queue
}

// Options TODO
type Options struct {
	ID []byte
}

// New TODO
func (dht *DHT) New(options *Options) *DHT {
	return nil
}

// Store TODO
func (dht *DHT) Store(data []byte) {

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
