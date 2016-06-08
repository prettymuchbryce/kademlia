package dht

import (
	"bytes"
	"crypto/rand"
	"errors"
	"sort"
	"sync"
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

var (
	errorValueNotFound = errors.New("Value not found")
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

// hashTable represents the hashtable state
type hashTable struct {
	// The ID of the local node
	ID []byte

	// Routing table a list of all known nodes in the network
	RoutingTable [][]*node // 160x20

	// The networking interface
	Networking networking

	mutex *sync.Mutex
}

func newHashTable() *hashTable {
	ht := &hashTable{}

	ht.mutex = &sync.Mutex{}

	for i := 0; i < b; i++ {
		ht.RoutingTable = append(ht.RoutingTable, []*node{})
	}

	return ht
}

func (ht *hashTable) listen() {
	go func() {
		for {
			msg := ht.Networking.getMessage()
			switch msg.Type {
			case messageTypeQueryFindNode:

			case messageTypeQueryFindValue:

			case messageTypeQueryStore:

			case messageTypeQueryPing:

			}
		}
	}()
}

func (ht *hashTable) getClosestContacts(num int, target []byte) *shortList {
	// First we need to build the list of adjacent indices to our target
	// in order
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

	leftToAdd := num

	// Next we select alpha contacts and add them to the short list
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

	return sl
}

// Iterate does an iterative search through the network. This can be done
// for multiple reasons. These reasons include:
//     iterativeStore - Used to store new information in the network.
//     iterativeFindNode - Used to bootstrap the network
//     iterativeFindValue - Used to find a value among the network given a key
func (ht *hashTable) iterate(t int, target []byte, data []byte) (value []byte, closest []*node, error error) {
	defer ht.mutex.Unlock()
	ht.mutex.Lock()

	sl := ht.getClosestContacts(alpha, target)

	// We keep track of nodes contacted so far. We don't contact the same node
	// twice.
	var contacted = make(map[string]bool)

	// We keep a reference to the closestNode. If after performing a search
	// we do not find a closer node, we stop searching.
	closestNode := sl.Nodes[0]

	for {
		queries := []*message{}
		// Next we send messages to the first (closest) alpha nodes in the
		// shortlist and wait for a response
		for i, node := range sl.Nodes {
			// Contact only alpha nodes
			if i >= alpha {
				break
			}

			// Don't contact nodes already contacted
			if contacted[string(node.ID)] == true {
				continue
			}

			contacted[string(node.ID)] = true
			query := &message{}
			query.Node = node

			switch t {
			case iterateFindNode:
				query.Type = messageTypeQueryFindNode
				queryData := &queryDataFindNode{}
				queryData.Target = target
				query.Data = queryData
			case iterateFindValue:
				query.Type = messageTypeQueryFindValue
				queryData := &queryDataFindValue{}
				queryData.Key = target
				query.Data = queryData
			case iterateStore:
				query.Type = messageTypeQueryStore
				queryData := &queryDataStore{}
				queryData.Key = target
				queryData.Data = data
				query.Data = queryData
			default:
				panic("Unknown iterate type")
			}

			queries = append(queries, query)
		}

		// Send the async queries and wait for a response
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
				for _, node := range responseData.Closest {
					ht.addNode(node)
				}
				sl.AppendUnique(responseData.Closest)
			case iterateFindValue:
				responseData := result.Data.(*responseDataFindValue)
				if responseData.Value != nil {
					return responseData.Value, nil, nil
				}
				for _, node := range responseData.Closest {
					ht.addNode(node)
				}
				sl.AppendUnique(responseData.Closest)
			case iterateStore:
				responseData := result.Data.(*responseDataFindNode)
				for _, node := range responseData.Closest {
					ht.addNode(node)
				}
				sl.AppendUnique(responseData.Closest)
			}
		}

		sort.Sort(sl)

		// If closestNode is unchanged then we are done
		if bytes.Compare(sl.Nodes[0].ID, closestNode.ID) == 0 {
			// We are done
			switch t {
			case iterateFindNode:
				return nil, sl.Nodes, nil
			case iterateFindValue:
				return nil, sl.Nodes, nil
			case iterateStore:
				var queries []*message
				for i, n := range sl.Nodes {
					if i >= k {
						return nil, nil, nil
					}

					query := &message{}
					query.Node = n
					query.Type = messageTypeQueryStore
					queryData := &queryDataStore{}
					queryData.Data = data
					queryData.Key = target
					query.Data = queryData
					queries = append(queries, query)
				}
				ht.Networking.sendMessages(queries)
				return nil, nil, nil
			}
		} else {
			closestNode = sl.Nodes[0]
		}
	}
}

// addNode adds a node into the appropriate k bucket
// we store these buckets in big-endian order so we look at the bits
// from right to left in order to find the appropriate bucket
func (ht *hashTable) addNode(node *node) {
	index := ht.getBucketIndexFromDifferingBit(ht.ID, node.ID)
	bucket := ht.RoutingTable[index]

	// Make sure node doesn't already exist
	for _, v := range bucket {
		if bytes.Compare(v.ID, node.ID) == 0 {
			return
		}
	}

	bucket = append(bucket, node)

	// TODO sort by recently seen

	// If there are more than k items in the bucket, remove
	// the last one
	// TODO The Kademlia paper suggests pinging the last node first, and
	// leaving it if it responds. That is - we have a preference for old contacts
	if len(bucket) > k {
		bucket = bucket[:len(bucket)-1]
	}

	ht.RoutingTable[index] = bucket
}

func (ht *hashTable) getBucketIndexFromDifferingBit(id1 []byte, id2 []byte) int {
	// Look at each byte from right to left
	for j := 0; j < len(id1); j++ {
		// xor the byte
		xor := id1[j] ^ id2[j]

		// check each bit on the xored result from left to right in order
		for i := 0; i < 8; i++ {
			if hasBit(xor, uint(i)) {
				byteIndex := j * 8
				bitIndex := i
				return b - (byteIndex + bitIndex) - 1
			}
		}
	}

	// the ids must be the same
	// should never happen
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

// Example:
// number:  1
// bits:    00000001
// pos:     01234567
func hasBit(n byte, pos uint) bool {
	pos = 7 - pos
	val := n & (1 << pos)
	return (val > 0)
}
