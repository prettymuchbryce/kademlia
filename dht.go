package main

import (
	"bytes"
	"crypto/sha1"
	"math"
	"sort"
	"time"

	b58 "github.com/jbenet/go-base58"
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

	// the maximum time to wait for a response from a node before discarding
	// it from the bucket
	tPingMax = 1

	// the maximum time to wait for a response to any message
	tMsgTimeout = 2
)

// DHT TODO
type DHT struct {
	ht         *hashTable
	options    *Options
	networking networking
	store      Store
	msgCounter int64
}

// Options TODO
type Options struct {
	ID             []byte
	IP             string
	Port           string
	BootstrapNodes []*NetworkNode
	ContactTimeout *time.Time
}

type chanQuery struct {
	query *message
	ch    chan (*message)
}

// NewDHT TODO
func NewDHT(store Store, options *Options) (*DHT, error) {
	dht := &DHT{}
	dht.options = options
	ht, err := newHashTable(options)
	if err != nil {
		return nil, err
	}
	dht.store = store
	dht.ht = ht
	dht.networking = &realNetworking{}
	return dht, nil
}

func (dht *DHT) getExpirationTime(key []byte) time.Time {
	if len(key) != 20 {
		panic("WTF")
	}
	bucket := getBucketIndexFromDifferingBit(key, dht.ht.Self.ID)
	var total int
	for i := 0; i < bucket; i++ {
		total += dht.ht.getTotalNodesInBucket(i)
	}
	closer := dht.ht.getAllNodesInBucketCloserThan(bucket, key)
	score := total + len(closer)

	if score == 0 {
		score = 1
	}

	if score > k {
		return time.Now().Add(time.Hour * 24)
	} else {
		day := time.Hour * 24
		seconds := day.Nanoseconds() * int64(math.Exp(float64(k/score)))
		dur := time.Second * time.Duration(seconds)
		return time.Now().Add(dur)
	}
}

// Store TODO
func (dht *DHT) Store(data []byte) (string, error) {
	sha := sha1.Sum(data)
	key := sha[:]
	expiration := dht.getExpirationTime(key)
	dht.store.Store(key, data, expiration, true)
	_, _, err := dht.iterate(iterateStore, key[:], data)
	if err != nil {
		return "", err
	}
	str := b58.Encode(key)
	return str, nil
}

// Get TODO
func (dht *DHT) Get(key string) ([]byte, bool, error) {
	keyBytes := b58.Decode(key)
	value, exists := dht.store.Retrieve(keyBytes)
	if !exists {
		var err error
		value, _, err = dht.iterate(iterateFindValue, keyBytes, nil)
		if err != nil {
			return nil, false, err
		}
		if value != nil {
			exists = true
		}
	}

	return value, exists, nil
}

func (dht *DHT) CreateSocket() error {
	ip := dht.options.IP
	port := dht.options.Port

	if ip == "" {
		ip = "0.0.0.0"
	}
	if port == "" {
		port = "3000"
	}

	netMsgInit()
	dht.networking.init()

	return dht.networking.createSocket(ip, port)
}

func (dht *DHT) Listen() error {
	go dht.listen()
	go dht.timers()
	return dht.networking.listen()
}

func (dht *DHT) Bootstrap() error {
	if len(dht.options.BootstrapNodes) > 0 {
		for _, bn := range dht.options.BootstrapNodes {
			node := newNode(bn)
			dht.addNode(node)
		}
	}
	_, _, err := dht.iterate(iterateFindNode, dht.ht.Self.ID, nil)
	return err
}

// Disconnect TODO
func (dht *DHT) Disconnect() error {
	return dht.networking.disconnect()
}

// Iterate does an iterative search through the network. This can be done
// for multiple reasons. These reasons include:
//     iterativeStore - Used to store new information in the network.
//     iterativeFindNode - Used to bootstrap the network.
//     iterativeFindValue - Used to find a value among the network given a key.
func (dht *DHT) iterate(t int, target []byte, data []byte) (value []byte, closest []*NetworkNode, err error) {
	sl := dht.ht.getClosestContacts(alpha, target, []*NetworkNode{})

	// We keep track of nodes contacted so far. We don't contact the same node
	// twice.
	var contacted = make(map[string]bool)

	// We keep a reference to the closestNode. If after performing a search
	// we do not find a closer node, we stop searching.
	if len(sl.Nodes) == 0 {
		return nil, nil, nil
	}

	closestNode := sl.Nodes[0]

	if t == iterateFindNode {
		bucket := getBucketIndexFromDifferingBit(target, dht.ht.Self.ID)
		dht.ht.resetRefreshTimeForBucket(bucket)
	}

	for {
		queries := []*chanQuery{}
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
			query.Sender = dht.ht.Self
			query.Receiver = node

			switch t {
			case iterateFindNode:
				query.Type = messageTypeQueryFindNode
				queryData := &queryDataFindNode{}
				queryData.Target = target
				query.Data = queryData
			case iterateFindValue:
				query.Type = messageTypeQueryFindValue
				queryData := &queryDataFindValue{}
				queryData.Target = target
				query.Data = queryData
			case iterateStore:
				query.Type = messageTypeQueryFindNode
				queryData := &queryDataFindNode{}
				queryData.Target = target
				query.Data = queryData
			default:
				panic("Unknown iterate type")
			}

			// Send the async queries and wait for a response
			ch, err := dht.networking.sendMessage(query, dht.msgCounter, true)
			dht.msgCounter++
			if err != nil {
				// Node was unreachable for some reason. We will remove it from
				// the shortlist, but keep it around in hopes that it might
				// come back online in the future.
				sl.RemoveNode(query.Receiver)
				continue
			}
			queries = append(queries, &chanQuery{query: query, ch: ch})
		}

		resultChan := make(chan (*message))
		for _, q := range queries {
			go func() {
				select {
				case result := <-q.ch:
					// TODO sanity check incoming messages ?
					// Make sure that the node id matches the one in the
					// query along with some other data. Otherwise it
					// becomes fairly easy to break the network with
					// some bad messages
					if result == nil {
						panic("This should never happen")
						// dht.networking.cancelResponse(q.query.ID)
					} else {
						dht.ht.markNodeAsSeen(result.Sender.ID)
						resultChan <- result
					}
				}
			}()
		}

		var results []*message
	Loop:
		for {
			select {
			case result := <-resultChan:
				results = append(results, result)
				if len(results) == len(queries) {
					close(resultChan)
					break Loop
				}
			case <-time.After(time.Second * tMsgTimeout):
				// TODO kill channel?
				close(resultChan)
				break Loop
			}
		}

		for _, result := range results {
			if result.Error != nil {
				sl.RemoveNode(result.Receiver)
				continue
			}
			switch t {
			case iterateFindNode:
				responseData := result.Data.(*responseDataFindNode)
				for _, n := range responseData.Closest {
					dht.addNode(newNode(n))
				}
				sl.AppendUniqueNetworkNodes(responseData.Closest)
			case iterateFindValue:
				responseData := result.Data.(*responseDataFindValue)
				// TODO When an iterativeFindValue succeeds, the initiator must
				// store the key/value pair at the closest node seen which did
				// not return the value.
				if responseData.Value != nil {
					return responseData.Value, nil, nil
				}
				for _, n := range responseData.Closest {
					dht.addNode(newNode(n))
				}
				sl.AppendUniqueNetworkNodes(responseData.Closest)
			case iterateStore:
				responseData := result.Data.(*responseDataFindNode)
				for _, n := range responseData.Closest {
					dht.addNode(newNode(n))
				}
				sl.AppendUniqueNetworkNodes(responseData.Closest)
			}
		}

		if len(sl.Nodes) == 0 {
			return nil, nil, nil
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
				for i, n := range sl.Nodes {
					if i >= k {
						return nil, nil, nil
					}

					query := &message{}
					query.Receiver = n
					query.Sender = dht.ht.Self
					query.Type = messageTypeQueryStore
					queryData := &queryDataStore{}
					queryData.Data = data
					queryData.Key = target
					query.Data = queryData
					dht.networking.sendMessage(query, dht.msgCounter, false)
					dht.msgCounter++
				}
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
func (dht *DHT) addNode(node *node) {
	index := getBucketIndexFromDifferingBit(dht.ht.Self.ID, node.ID)

	// Make sure node doesn't already exist
	// If it does, mark it as seen
	if dht.ht.doesNodeExistInBucket(index, node.ID) {
		dht.ht.markNodeAsSeen(node.ID)
		return
	}

	dht.ht.mutex.Lock()
	defer dht.ht.mutex.Unlock()

	bucket := dht.ht.RoutingTable[index]

	if len(bucket) == k {
		// If the bucket is full we need to ping the first node to find out
		// if it responds back in a reasonable amount of time. If not -
		// we may remove it
		n := bucket[0].NetworkNode
		query := &message{}
		query.Receiver = n
		query.Sender = dht.ht.Self
		query.Type = messageTypeQueryPing
		ch, err := dht.networking.sendMessage(query, dht.msgCounter, true)
		dht.msgCounter++
		if err != nil {
			bucket = append(bucket, node)
			bucket = bucket[1:]
		} else {
			select {
			case <-ch:
				return
			case <-time.After(time.Second * tPingMax):
				bucket = bucket[1:]
				bucket = append(bucket, node)
			}
		}
	} else {
		bucket = append(bucket, node)
	}

	dht.ht.RoutingTable[index] = bucket
}

func (dht *DHT) timers() {
	t := time.NewTicker(time.Nanosecond)
	for {
		select {
		case <-t.C:
			for i := 0; i < b; i++ {
				if time.Since(dht.ht.getRefreshTimeForBucket(i)) > time.Second*tRefresh {
					id := dht.ht.getRandomIDFromBucket(k)
					dht.iterate(iterateFindNode, id, nil)
				}
			}

			keys := dht.store.GetAllKeysForRefresh()
			for _, key := range keys {
				keyBytes := b58.Decode(key)
				value, _ := dht.store.Retrieve(keyBytes)
				dht.iterate(iterateStore, keyBytes, value)
			}

			dht.store.ExpireKeys()
		case <-dht.networking.getDisconnect():
			t.Stop()
			dht.networking.timersFin()
			return
		}
	}
}

func (dht *DHT) listen() {
	for {
		select {
		case msg := <-dht.networking.getMessage():
			if msg == nil {
				// Disconnected
				dht.networking.messagesFin()
				return
			}
			switch msg.Type {
			case messageTypeQueryFindNode:
				data := msg.Data.(*queryDataFindNode)
				dht.addNode(newNode(msg.Sender))
				closest := dht.ht.getClosestContacts(k, data.Target, []*NetworkNode{msg.Sender})
				response := &message{IsResponse: true}
				response.Sender = dht.ht.Self
				response.Receiver = msg.Sender
				response.Type = messageTypeResponseFindNode
				responseData := &responseDataFindNode{}
				responseData.Closest = closest.Nodes
				response.Data = responseData
				dht.networking.sendMessage(response, msg.ID, false)
			case messageTypeQueryFindValue:
				data := msg.Data.(*queryDataFindValue)
				dht.addNode(newNode(msg.Sender))
				value, exists := dht.store.Retrieve(data.Target)
				response := &message{IsResponse: true}
				response.ID = msg.ID
				response.Receiver = msg.Sender
				response.Sender = dht.ht.Self
				response.Type = messageTypeResponseFindValue
				responseData := &responseDataFindValue{}
				if exists {
					responseData.Value = value
				} else {
					closest := dht.ht.getClosestContacts(k, data.Target, []*NetworkNode{msg.Sender})
					responseData.Closest = closest.Nodes
				}
				response.Data = responseData
				dht.networking.sendMessage(response, msg.ID, false)
			case messageTypeQueryStore:
				data := msg.Data.(*queryDataStore)
				dht.addNode(newNode(msg.Sender))
				expiration := dht.getExpirationTime(data.Key)
				dht.store.Store(data.Key, data.Data, expiration, false)
			case messageTypeQueryPing:
				response := &message{IsResponse: true}
				response.Sender = dht.ht.Self
				response.Receiver = msg.Sender
				response.Type = messageTypeResponsePing
				dht.networking.sendMessage(response, msg.ID, false)
			}
		case <-dht.networking.getDisconnect():
			dht.networking.messagesFin()
			return
		}
	}
}
