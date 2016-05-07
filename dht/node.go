package dht

import (
	"crypto/rand"
	"math/big"
	"net"
	"sort"
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

func (n shortList) RemoveNode(node *node) {
	for i := 0; i < n.Len(); i++ {
		if n.Nodes[i] == node {
			n.Nodes = append(n.Nodes[:i], n.Nodes[i+1:]...)
			return
		}
	}
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
	ID []byte

	// Routing table a list of all known nodes in the network
	RoutingTable [][]*node // 160x20

	// The bootstrap node
	Bootstrap *node

	// The networking interface
	Networking networking
}

func (ht *hashTable) iterativeStore(id []byte, data []byte) {

}

// store
// find value
// find node
func (ht *hashTable) iterate(t int, target []byte, data []byte) (error, []byte) {
	// First we need to build the list of adjacent indices in order
	index := ht.getBucketIndexFromDifferingBit(ht.ID, target)
	indexList := []int{index}
	i := index - 1
	j := index + 1
	for len(indexList) < b {
		if j < b {
			indexList = append(indexList, j)
		}
		if i >= 0 {
			indexList = append(indexList, i)
		}
		i--
		j++
	}

	sl := &shortList{}
	leftToAdd := alpha

	// Next we select alpha contacts
	for leftToAdd > 0 && len(indexList) > 0 {
		index, indexList = indexList[0], indexList[1:]
		len := len(ht.RoutingTable[index])
		if len < leftToAdd {
			sl.Nodes = append(sl.Nodes, ht.RoutingTable[index]...)
		} else {
			sl.Nodes = append(sl.Nodes, ht.RoutingTable[index][:leftToAdd]...)
		}
	}

	sort.Sort(sl)

	closestNode := sl.Nodes[0]
	queries := []*query{}
	// Next we send messages to the nodes in the shortlist and wait for a
	// response
	for _, node := range sl.Nodes {
		query := &query{}
		query.Node = node

		switch t {
		case iterateFindNode:
			query.Type = messageTypeFindNode
			queryData := &queryDataFindNode{}
			queryData.Target = target
			query.Data = queryData
		case iterateFindValue:
			query.Type = messageTypeFindValue
			queryData := &queryDataFindValue{}
			queryData.Key = target
			query.Data = queryData
		case iterateStore:
			query.Type = messageTypeStore
			queryData := &queryDataStore{}
			queryData.Key = target
			queryData.Data = data
			query.Data = queryData
		default:
			panic("Unknown iterate type")
		}

		queries = append(queries, query)
	}

	channel := ht.Networking.sendMessages(queries)
	results := <-channel

	for _, result := range results {
		if result.Error != nil {
			sl.RemoveNode(result.Node)
			continue
		}
		switch t {
		case iterateFindNode:
			responseData := result.Data.(*responseDataFindNode)
			sl = sl.append(sl, responseData.Closest...)
		case iterateFindValue:
			responseData := result.Data.(*responseDataFindValue)
			sl = sl.append(sl, responseData)
		case iterateStore:
		}
	}

}

// addNode adds a node into the appropriate k bucket
// we store these buckets in big-endian order so we look at the bits
// from right to left in order to find the appropriate bucket
func (ht *hashTable) addNode(node *node) {

	index := ht.getBucketIndexFromDifferingBit(ht.ID, node.ID)
	bucket := ht.RoutingTable[index]
	bucket = append(bucket, node)

	// TODO sort by recently seen

	// If there are more than k items in the bucket, remove
	// the last one
	if len(bucket) > k {
		bucket = bucket[:len(bucket)-1]
	}
}

func (ht *hashTable) getBucketIndexFromDifferingBit(id1 []byte, id2 []byte) int {
	// Look at each byte from right to left
	for j := len(id1); j <= 0; j-- {
		// xor the byte
		xor := id1[j] ^ id2[j]

		// check each bit on the xored result from right to left in order
		for i := 7; i >= 0; i-- {
			if hasBit(xor, uint(i)) {
				byteIndex := j * 8
				bitIndex := i
				return b - (byteIndex + bitIndex)
			}
		}
	}

	// the ids must be the same
	return 0
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
