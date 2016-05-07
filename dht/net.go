package dht

const (
	messageTypePing      = "PING"
	messageTypeStore     = "STORE"
	messageTypeFindNode  = "FIND_NODE"
	messageTypeFindValue = "FIND_VALUE"
)

type query struct {
	Type string
	Node *node
	Data interface{}
}

type queryDataFindNode struct {
	Target []byte
}

type queryDataFindValue struct {
	Key []byte
}

type queryDataStore struct {
	Data []byte
	Key  []byte
}

type response struct {
	Node  *node
	Error error
	Data  interface{}
}

type responseDataFindNode struct {
	Closest []*node
}

type responseDataFindValue struct {
	Closest []*node
	Value   []byte
}

type responseDataStore struct {
	Success bool
}

// Network TODO
type networking interface {
	sendMessages([]*query) chan ([]*response)
}

type mockNetworking struct {
	recv chan ([]*query)
	send chan ([]*response)
}

func newMockNetworking() *mockNetworking {
	net := &mockNetworking{}
	net.recv = make(chan ([]*query))
	net.send = make(chan ([]*response))
	return net
}

func (net *mockNetworking) sendMessages(q []*query) chan ([]*response) {
	net.recv <- q
	return net.send
}

//
// Listen(net string, laddr *net.UDPAddr) (Connection, error)

//
// // Connection TODO
// type Connection interface {
// 	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
// 	Close() error
// }
//
// // MockConnection TODO
// type MockConnection struct {
// }
//
// // MockNetwork TODO
// type MockNetwork struct {
// }
//
// // NetworkImpl TODO
// type NetworkImpl struct {
// }
//
// type queryResponseChannelData struct {
// 	response *response
// 	err      error
// }
//
// func doQuery(nodes []*Node, query *query) chan *queryResponseChannelData {
// 	c := make(chan *queryResponseChannelData)
//
// 	for i := 0; i < len(nodes); i++ {
// 		go func(node *Node) {
//
// 		}(nodes[i])
// 	}
//
// 	return c
// }
