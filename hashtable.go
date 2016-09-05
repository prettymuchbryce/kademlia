package kademlia

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
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
	// Nodes within buckets are sorted by least recently seen e.g.
	// [ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ][ ]
	//  ^                                                           ^
	//  └ Least recently seen                    Most recently seen ┘
	RoutingTable [][]*node // 160x20

	mutex *sync.Mutex

	refreshMap [b]time.Time
}

func newHashTable(options *Options) (*hashTable, error) {
	ht := &hashTable{}

	rand.Seed(time.Now().UnixNano())

	ht.mutex = &sync.Mutex{}
	ht.Self = &NetworkNode{}

	if options.ID != nil {
		ht.Self.ID = options.ID
	} else {
		id, err := newID()
		if err != nil {
			return nil, err
		}
		ht.Self.ID = id
	}

	if options.IP == "" || options.Port == "" {
		return nil, errors.New("Port and IP required")
	}

	err := ht.setSelfAddr(options.IP, options.Port)
	if err != nil {
		return nil, err
	}

	for i := 0; i < b; i++ {
		ht.resetRefreshTimeForBucket(i)
	}

	for i := 0; i < b; i++ {
		ht.RoutingTable = append(ht.RoutingTable, []*node{})
	}

	return ht, nil
}

func (ht *hashTable) setSelfAddr(ip string, port string) error {
	ht.Self.IP = net.ParseIP(ip)
	p, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	ht.Self.Port = p
	return nil
}

func (ht *hashTable) resetRefreshTimeForBucket(bucket int) {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	ht.refreshMap[bucket] = time.Now()
}

func (ht *hashTable) getRefreshTimeForBucket(bucket int) time.Time {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	return ht.refreshMap[bucket]
}

func (ht *hashTable) markNodeAsSeen(node []byte) {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	index := getBucketIndexFromDifferingBit(ht.Self.ID, node)
	bucket := ht.RoutingTable[index]
	nodeIndex := -1
	for i, v := range bucket {
		if bytes.Compare(v.ID, node) == 0 {
			nodeIndex = i
			break
		}
	}
	if nodeIndex == -1 {
		panic(errors.New("Tried to mark nonexistent node as seen"))
	}

	n := bucket[nodeIndex]
	bucket = append(bucket[:nodeIndex], bucket[nodeIndex+1:]...)
	bucket = append(bucket, n)
	ht.RoutingTable[index] = bucket
}

func (ht *hashTable) doesNodeExistInBucket(bucket int, node []byte) bool {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	for _, v := range ht.RoutingTable[bucket] {
		if bytes.Compare(v.ID, node) == 0 {
			return true
		}
	}
	return false
}

func (ht *hashTable) getClosestContacts(num int, target []byte, ignoredNodes []*NetworkNode) *shortList {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	// First we need to build the list of adjacent indices to our target
	// in order
	index := getBucketIndexFromDifferingBit(ht.Self.ID, target)
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
				if bytes.Compare(ht.RoutingTable[index][i].ID, ignoredNodes[j].ID) == 0 {
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

func (ht *hashTable) removeNode(ID []byte) {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()

	index := getBucketIndexFromDifferingBit(ht.Self.ID, ID)
	bucket := ht.RoutingTable[index]

	for i, v := range bucket {
		if bytes.Compare(v.ID, ID) == 0 {
			bucket = append(bucket[:i], bucket[i+1:]...)
		}
	}

	ht.RoutingTable[index] = bucket
}

func (ht *hashTable) getAllNodesInBucketCloserThan(bucket int, id []byte) [][]byte {
	b := ht.RoutingTable[bucket]
	var nodes [][]byte
	for _, v := range b {
		d1 := ht.getDistance(id, ht.Self.ID)
		d2 := ht.getDistance(id, v.ID)

		result := d1.Sub(d1, d2)
		if result.Sign() > -1 {
			nodes = append(nodes, v.ID)
		}
	}

	return nodes
}

func (ht *hashTable) getTotalNodesInBucket(bucket int) int {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	return len(ht.RoutingTable[bucket])
}

func (ht *hashTable) getDistance(id1 []byte, id2 []byte) *big.Int {
	var dst [k]byte
	for i := 0; i < k; i++ {
		dst[i] = id1[i] ^ id2[i]
	}
	ret := big.NewInt(0)
	return ret.SetBytes(dst[:])
}

func (ht *hashTable) getRandomIDFromBucket(bucket int) []byte {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	// Set the new ID to to be equal in every byte up to
	// the byte of the first differing bit in the bucket

	byteIndex := bucket / 8
	var id []byte
	for i := 0; i < byteIndex; i++ {
		id = append(id, ht.Self.ID[i])
	}
	differingBitStart := bucket % 8

	var firstByte byte
	// check each bit from left to right in order
	for i := 0; i < 8; i++ {
		// Set the value of the bit to be the same as the ID
		// up to the differing bit. Then begin randomizing
		var bit bool
		if i < differingBitStart {
			bit = hasBit(ht.Self.ID[byteIndex], uint(i))
		} else {
			bit = rand.Intn(2) == 1
		}

		if bit {
			firstByte += byte(math.Pow(2, float64(7-i)))
		}
	}

	id = append(id, firstByte)

	// Randomize each remaining byte
	for i := byteIndex + 1; i < 20; i++ {
		randomByte := byte(rand.Intn(256))
		id = append(id, randomByte)
	}

	return id
}

func getBucketIndexFromDifferingBit(id1 []byte, id2 []byte) int {
	// Look at each byte from left to right
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
	// this should only happen during bootstrapping
	return 0
}

func (ht *hashTable) totalNodes() int {
	ht.mutex.Lock()
	defer ht.mutex.Unlock()
	var total int
	for _, v := range ht.RoutingTable {
		total += len(v)
	}
	return total
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
