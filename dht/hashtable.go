package dht

import (
	"bytes"
	"crypto/rand"
	"errors"
	"net"
	"sort"
	"strconv"
	"sync"
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

// hashTable represents the hashtable state
type hashTable struct {
	// The ID of the local node
	Self *NetworkNode

	// Routing table a list of all known nodes in the network
	RoutingTable [][]*node // 160x20

	mutex *sync.Mutex
}

func newHashTable(options *Options) (*hashTable, error) {
	ht := &hashTable{}

	ht.mutex = &sync.Mutex{}
	ht.Self = &NetworkNode{}

	if options.ID != nil {
		ht.Self.ID = options.ID
	} else {
		id, err := newID()
		if err != nil {
			panic(err)
		}
		ht.Self.ID = id
	}

	if options.IP == "" || options.Port == "" {
		// TODO don't panic, bubble up.
		return nil, errors.New("Port and IP required")
	}

	ht.Self.IP = net.ParseIP(options.IP)
	port, err := strconv.Atoi(options.Port)
	if err != nil {
		return nil, err
	}
	ht.Self.Port = port

	for i := 0; i < b; i++ {
		ht.RoutingTable = append(ht.RoutingTable, []*node{})
	}

	return ht, nil
}

func (ht *hashTable) getClosestContacts(num int, target []byte, ignoredNodes []*NetworkNode) *shortList {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	// First we need to build the list of adjacent indices to our target
	// in order
	index := ht.getBucketIndexFromDifferingBit(ht.Self.ID, target)
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
		bucketContacts := len(ht.RoutingTable[index])
		for i := 0; i < bucketContacts; i++ {
			ignored := false
			for j := 0; j < len(ignoredNodes); j++ {
				if bytes.Compare(ignoredNodes[j].ID, ignoredNodes[j].ID) == 0 {
					ignored = true
				}
			}
			if !ignored {
				sl.AppendUnique([]*node{ht.RoutingTable[index][i]})
				leftToAdd--
				if leftToAdd == 0 {
					break
				}
			}
		}
	}

	sort.Sort(sl)

	return sl
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
