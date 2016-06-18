package dht

import (
	"bytes"
	"math/big"
)

// node represents a node in the network
type node struct {
	// ID is a 20 byte unique identifier
	// ID []byte
	//
	// // IP is the IPv4 address of the node
	// IP net.IP
	//
	// // Port is the port of the node
	// Port int
	*NetworkNode

	// LastSeen is a unix timestamp denoting the last time the node was seen
	// k-buckets are sorted by LastSeen
	LastSeen int64

	// RTT is a list of times in milliseconds that the node took to respond
	// to requests. This is used to determine how latent the node is
	RTT []int
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

func areBytesEqual(b1 []byte, b2 []byte) bool {
	for k := range b1 {
		if b1[k] != b2[k] {
			return false
		}
	}

	return true
}

func (n *shortList) RemoveNode(node *NetworkNode) {
	for i := 0; i < n.Len(); i++ {
		if areBytesEqual(n.Nodes[i].ID, node.ID) {
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
