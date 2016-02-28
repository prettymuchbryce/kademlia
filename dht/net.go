package dht

import "net"

// Network TODO
type Network interface {
	Listen(net string, laddr *net.UDPAddr) (Connection, error)
}

// Connection TODO
type Connection interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	Close() error
}

// MockConnection TODO
type MockConnection struct {
}

// MockNetwork TODO
type MockNetwork struct {
}

// NetworkImpl TODO
type NetworkImpl struct {
}
