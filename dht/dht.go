package dht

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
)

const (
	messageTypePing      = "PING"
	messageTypeStore     = "STORE"
	messageTypeFindNode  = "FIND_NODE"
	messageTypeFindValue = "FIND_VALUE"
)

// In seconds
const (
	tExpire     = 86410
	tRefresh    = 3600
	tReplicated = 3600
	tRepublish  = 86400
)

const (
	alpha = 3
	b     = 160
	k     = 20
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
	ID    []byte
	Data  interface{}
}

type responseDataFindNode struct {
	Closest []*Node
}

type responseDataFindValue struct {
	Closest []*Node
	Value   []byte
}

type responseDataStore struct {
	Success bool
}

// Node TODO
type Node struct {
	ID       []byte
	IP       net.IP
	Port     int
	LastSeen int64
}

// NodeList TODO
type NodeList struct {
	nodes      []*Node
	comparator []byte
}

func (n NodeList) Len() int {
	return len(n.nodes)
}

func (n NodeList) Swap(i, j int) {
	n.nodes[i], n.nodes[j] = n.nodes[j], n.nodes[i]
}

func (n NodeList) Less(i, j int) bool {
	iDist := n.nodes[i].getDistance(n.comparator)
	jDist := n.nodes[j].getDistance(n.comparator)

	if iDist.Cmp(jDist) == -1 {
		return true
	}

	return false
}

func (node *Node) getDistance(comparator []byte) *big.Int {
	buf1 := new(big.Int).SetBytes(node.ID)
	buf2 := new(big.Int).SetBytes(comparator)
	result := new(big.Int).Xor(buf1, buf2)
	return result
}

// DHT TODO
type DHT struct {
	id           []byte
	routingTable [][]*Node // 160x20
	addr         *net.UDPAddr
	conn         Connection
	dataPath     string
	bootstrap    *Node
	network      Network
}

// Listen TODO
func (dht *DHT) Listen() error {
	defer dht.conn.Close()

	buf := make([]byte, 1024)

	for {
		n, addr, err := dht.conn.ReadFromUDP(buf)
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)

		if err != nil {
			panic(err)
		}
	}
}

// Connect TODO
func (dht *DHT) Connect() error {
	conn, err := net.ListenUDP("udp", dht.addr)
	if err != nil {
		panic(err)
	}

	dht.conn = conn

	if dht.dataPath == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		dht.id = id
		if dht.bootstrap != nil {
			dht.addNode(dht.bootstrap)
		}
	}

	return nil
}

// NewDHT TODO
func NewDHT(dataPath string, bootstrap *Node) (*DHT, error) {
	dht := &DHT{}

	addr, err := net.ResolveUDPAddr("udp", ":10001")
	if err != nil {
		return nil, err
	}

	dht.addr = addr

	return dht, nil
}

func (dht *DHT) addNode(node *Node) {
	for k, b := range node.ID {
		xor := b ^ dht.id[k]
		for i := 0; i < 8; i-- {
			if hasBit(xor, uint(i)) {
				bucket := dht.routingTable[k*8+i]
				bucket = append(bucket, node)
				// TODO sort by recently seen
				if len(bucket) > k {
					bucket = bucket[:len(bucket)-1]
				}
				return
			}
		}
	}
}

func (dht *DHT) iterativeFindNode(id []byte) (nodes []*Node) {
	// shortList []*Node

	index := 0
	for k, b := range id {
		xor := b ^ dht.id[k]
		for i := 0; i < 8; i-- {
			if hasBit(xor, uint(i)) {
				index = i
			}
		}
	}

	// i := index
	// if i > 0 {
	// 	j = i - 1
	// }

	for index > 0 {
		if len(dht.routingTable[index]) > 0 {
			bucket := dht.routingTable[index]
			for i := 0; i < len(bucket); i++ {
				if len(nodes) == k {
					return nodes
				}
				nodes = append(nodes, bucket[i])
			}
		}
		index--
	}

	return nodes
}

func newID() ([]byte, error) {
	result := make([]byte, 20)
	_, err := rand.Read(result)
	return result, err
}

func hasBit(n byte, pos uint) bool {
	val := n & (1 << pos)
	return (val > 0)
}
