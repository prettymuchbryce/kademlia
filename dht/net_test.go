package dht

type mockNetworking struct {
	recv chan ([]*message)
	send chan ([]*message)
}

func newMockNetworking() *mockNetworking {
	net := &mockNetworking{}
	return net
}

func (net *mockNetworking) listen() error {
	return nil
}

func (net *mockNetworking) createSocket(host string, port string) error {
	return nil
}

func (net *mockNetworking) init() {
	net.recv = make(chan ([]*message))
	net.send = make(chan ([]*message))
}

func (net *mockNetworking) getMessage() *message {
	return nil
}

func (net *mockNetworking) sendMessages(q []*message) []*message {
	net.recv <- q
	return <-net.send
}
