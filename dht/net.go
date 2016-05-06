package dht

const (
	messageTypePing      = "PING"
	messageTypeStore     = "STORE"
	messageTypeFindNode  = "FIND_NODE"
	messageTypeFindValue = "FIND_VALUE"
)

type query struct {
	ID    []byte
	Type  string
	Token int32
	Data  interface{}
}

type queryDataFindNode struct {
	Target []byte
}

type queryDataFindValue struct {
	Key int32
}

type queryDataStore struct {
	Data []byte
	Key  []byte
}

type response struct {
	Token int64
	Error error
	ID    []byte
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
type network interface {
	sendMessages([]*query) chan ([]*response)
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
