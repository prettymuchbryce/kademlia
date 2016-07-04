package dht

import (
	"bytes"
	"math/big"
	"net"
)

// NetworkNode is the over-the-wire representation of a node
type NetworkNode struct {
	// ID is a 20 byte unique identifier
	ID []byte

	// IP is the IPv4 address of the node
	IP net.IP

	// Port is the port of the node
	Port int
}

// node represents a node in the network locally
// a separate struct due to the fact that we may want to add some metadata
// here later such as RTT, or LastSeen time
type node struct {
	*NetworkNode
}

func newNode(networkNode *NetworkNode) *node {
	n := &node{}
	n.NetworkNode = networkNode
	return n
}

// nodeList is used in order to sort a list of arbitrary nodes against a
// comparator. These nodes are sorted by xor distance
type shortList struct {
	// Nodes are a list of nodes to be compared
	Nodes []*NetworkNode

	// Comparator is the ID to compare to
	Comparator []byte
}

func (n *shortList) RemoveNode(node *NetworkNode) {
	for i := 0; i < n.Len(); i++ {
		if bytes.Compare(n.Nodes[i].ID, node.ID) == 0 {
			n.Nodes = append(n.Nodes[:i], n.Nodes[i+1:]...)
			return
		}
	}
}

func (n *shortList) AppendUniqueNetworkNodes(nodes []*NetworkNode) {
	for _, vv := range nodes {
		exists := false
		for _, v := range n.Nodes {
			if bytes.Compare(v.ID, vv.ID) == 0 {
				exists = true
			}
		}
		if !exists {
			n.Nodes = append(n.Nodes, vv)
		}
	}
}

func (n *shortList) AppendUnique(nodes []*node) {
	for _, vv := range nodes {
		exists := false
		for _, v := range n.Nodes {
			if bytes.Compare(v.ID, vv.ID) == 0 {
				exists = true
			}
		}
		if !exists {
			n.Nodes = append(n.Nodes, vv.NetworkNode)
		}
	}
}

func (n *shortList) Len() int {
	return len(n.Nodes)
}

func (n *shortList) Swap(i, j int) {
	n.Nodes[i], n.Nodes[j] = n.Nodes[j], n.Nodes[i]
}

func (n *shortList) Less(i, j int) bool {
	iDist := getDistance(n.Nodes[i].ID, n.Comparator)
	jDist := getDistance(n.Nodes[j].ID, n.Comparator)

	if iDist.Cmp(jDist) == -1 {
		return true
	}

	return false
}

func getDistance(id1 []byte, id2 []byte) *big.Int {
	buf1 := new(big.Int).SetBytes(id1)
	buf2 := new(big.Int).SetBytes(id2)
	result := new(big.Int).Xor(buf1, buf2)
	return result
}
