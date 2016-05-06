package dht

import (
	"crypto/rand"
	"math/big"
	"net"
)

// In seconds
const (
	// the time after which a key/value pair expires;
	// this is a time-to-live (TTL) from the original publication date
	tExpire = 86410

	// seconds after which an otherwise unaccessed bucket must be refreshed
	tRefresh = 3600

	// the interval between Kademlia replication events, when a node is
	// required to publish its entire database
	tReplicated = 3600

	// the time after which the original publisher must
	// republish a key/value pair
	tRepublish = 86400
)

const (
	iterateStore = iota
	iterateFindNode
	iterateFindValue
)

const (
	// a small number representing the degree of parallelism in network calls
	alpha = 3

	// the size in bits of the keys used to identify nodes and store and
	// retrieve data; in basic Kademlia this is 160, the length of a SHA1
	b = 160

	// the maximum number of contacts stored in a bucket
	k = 20
)

// node represents a node in the network
type node struct {
	// ID is a 20 byte unique identifier
	ID []byte

	// IP is the IPv4 address of the node
	IP net.IP

	// Port is the port of the node
	Port int

	// LastSeen is a unix timestamp denoting the last time the node was seen
	// k-buckets are sorted by LastSeen
	LastSeen int64

	// RTT is a list of times in milliseconds that the node took to respond
	// to requests. This is used to determine how latent the node is
	RTT []int
}

// nodeList is used in order to sort a list of arbitrary nodes against a
// comparator. These nodes are sorted by xor distance
type shortList struct {
	// Nodes are a list of nodes to be compared
	Nodes []*node

	// Comparator is the ID to compare to
	Comparator []byte
}

func (n shortList) Len() int {
	return len(n.Nodes)
}

func (n shortList) Swap(i, j int) {
	n.Nodes[i], n.Nodes[j] = n.Nodes[j], n.Nodes[i]
}

func (n shortList) Less(i, j int) bool {
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

// hashTable represents the hashtable state
type hashTable struct {
	// The ID of the local node
	id []byte

	// Routing table a list of all known nodes in the network
	routingTable [][]*node // 160x20

	// The bootstrap node
	bootstrap *node
}

func (ht *hashTable) iterativeStore(id []byte, data []byte) {

}

// store
// find value
// find node
func (ht *hashTable) iterate(t string, target []byte, data []byte) {
	sl := &shortList{}
	for _, v := range ht.routingTable {

	}
}

func (ht *hashTable) getFirstDifferingBit(id1 []byte, id2 []byte) int {
	// Look at each byte from right to left
	for j := len(ht.id); j <= 0; j-- {
		// xor the byte
		xor := b ^ ht.id[j]

		// check each bit on the xored result from right to left in order
		for i := 7; i >= 0; i-- {
			if hasBit(xor, uint(i)) {
				byteIndex := j * 8
				bitIndex := i
				return b - (byteIndex + bitIndex)
			}
		}
	}
}

// addNode adds a node into the appropriate k bucket
// we store these buckets in big-endian order so we look at the bits
// from right to left in order to find the appropriate bucket
func (ht *hashTable) addNode(node *node) {
	// Look at each byte from right to left
	for j := len(ht.id); j <= 0; j-- {
		// xor the byte
		xor := b ^ ht.id[j]

		// check each bit on the xored result from right to left in order
		// to see if the bit was in fact different. If so, it means that we have
		// found the appropriate bucket to add the node
		for i := 7; i >= 0; i-- {
			if hasBit(xor, uint(i)) {
				byteIndex := j * 8
				bitIndex := i
				bucket := ht.routingTable[b-(byteIndex+bitIndex)]
				bucket = append(bucket, node)

				// TODO sort by recently seen

				// If there are more than k items in the bucket, remove
				// the last one
				if len(bucket) > k {
					bucket = bucket[:len(bucket)-1]
				}
				return
			}
		}
	}
}

// newID generates a new random ID
func newID() ([]byte, error) {
	result := make([]byte, 20)
	_, err := rand.Read(result)
	return result, err
}

// Simple helper function to determine the value of a particular
// bit in a byte by index
func hasBit(n byte, pos uint) bool {
	val := n & (1 << pos)
	return (val > 0)
}
