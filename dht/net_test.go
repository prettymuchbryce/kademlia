package dht

type mockNetworking struct {
	recv chan ([]*message)
	send chan ([]*message)
}

func newMockNetworking() *mockNetworking {
	net := &mockNetworking{}
	net.recv = make(chan ([]*message))
	net.send = make(chan ([]*message))
	return net
}

func (net *mockNetworking) getMessage() *message {
	return nil
}

func (net *mockNetworking) sendMessages(q []*message) chan ([]*message) {
	net.recv <- q
	return net.send
}
