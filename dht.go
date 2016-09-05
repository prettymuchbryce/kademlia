package kademlia

import (
	"bytes"
	"errors"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	b58 "github.com/jbenet/go-base58"
)

// DHT represents the state of the local node in the distributed hash table
type DHT struct {
	ht         *hashTable
	options    *Options
	networking networking
	store      Store
}

// Options contains configuration options for the local node
type Options struct {
	ID []byte

	// The local IPv4 or IPv6 address
	IP string

	// The local port to listen for connections on
	Port string

	// Whether or not to use the STUN protocol to determine public IP and Port
	// May be necessary if the node is behind a NAT
	UseStun bool

	// Specifies the the host of the STUN server. If left empty will use the
	// default specified in go-stun.
	StunAddr string

	// A logger interface
	Logger log.Logger

	// The nodes being used to bootstrap the network. Without a bootstrap
	// node there is no way to connect to the network. NetworkNodes can be
	// initialized via dht.NewNetworkNode()
	BootstrapNodes []*NetworkNode

	// The time after which a key/value pair expires;
	// this is a time-to-live (TTL) from the original publication date
	TExpire time.Duration

	// Seconds after which an otherwise unaccessed bucket must be refreshed
	TRefresh time.Duration

	// The interval between Kademlia replication events, when a node is
	// required to publish its entire database
	TReplicate time.Duration

	// The time after which the original publisher must
	// republish a key/value pair. Currently not implemented.
	TRepublish time.Duration

	// The maximum time to wait for a response from a node before discarding
	// it from the bucket
	TPingMax time.Duration

	// The maximum time to wait for a response to any message
	TMsgTimeout time.Duration
}

// NewDHT initializes a new DHT node. A store and options struct must be
// provided.
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

	store.Init()

	if options.TExpire == 0 {
		options.TExpire = time.Second * 86410
	}

	if options.TRefresh == 0 {
		options.TRefresh = time.Second * 3600
	}

	if options.TReplicate == 0 {
		options.TReplicate = time.Second * 3600
	}

	if options.TRepublish == 0 {
		options.TRepublish = time.Second * 86400
	}

	if options.TPingMax == 0 {
		options.TPingMax = time.Second * 1
	}

	if options.TMsgTimeout == 0 {
		options.TMsgTimeout = time.Second * 2
	}

	return dht, nil
}

func (dht *DHT) getExpirationTime(key []byte) time.Time {
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
		return time.Now().Add(dht.options.TExpire)
	}

	day := dht.options.TExpire
	seconds := day.Nanoseconds() * int64(math.Exp(float64(k/score)))
	dur := time.Second * time.Duration(seconds)
	return time.Now().Add(dur)
}

// Store stores data on the network. This will trigger an iterateStore message.
// The base58 encoded identifier will be returned if the store is successful.
func (dht *DHT) Store(data []byte) (id string, err error) {
	key := dht.store.GetKey(data)
	expiration := dht.getExpirationTime(key)
	replication := time.Now().Add(dht.options.TReplicate)
	dht.store.Store(key, data, replication, expiration, true)
	_, _, err = dht.iterate(iterateStore, key[:], data)
	if err != nil {
		return "", err
	}
	str := b58.Encode(key)
	return str, nil
}

// Get retrieves data from the networking using key. Key is the base58 encoded
// identifier of the data.
func (dht *DHT) Get(key string) (data []byte, found bool, err error) {
	keyBytes := b58.Decode(key)
	value, exists := dht.store.Retrieve(keyBytes)

	if len(keyBytes) != k {
		return nil, false, errors.New("Invalid key")
	}

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

// NumNodes returns the total number of nodes stored in the local routing table
func (dht *DHT) NumNodes() int {
	return dht.ht.totalNodes()
}

// GetSelfID returns the base58 encoded identifier of the local node
func (dht *DHT) GetSelfID() string {
	str := b58.Encode(dht.ht.Self.ID)
	return str
}

// GetNetworkAddr returns the publicly accessible IP and Port of the local
// node
func (dht *DHT) GetNetworkAddr() string {
	return dht.networking.getNetworkAddr()
}

// CreateSocket attempts to open a UDP socket on the port provided to options
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
	dht.networking.init(dht.ht.Self)

	publicHost, publicPort, err := dht.networking.createSocket(ip, port, dht.options.UseStun, dht.options.StunAddr)
	if err != nil {
		return err
	}

	if dht.options.UseStun {
		dht.ht.setSelfAddr(publicHost, publicPort)
	}

	return nil
}

// Listen begins listening on the socket for incoming messages
func (dht *DHT) Listen() error {
	if !dht.networking.isInitialized() {
		return errors.New("socket not created")
	}
	go dht.listen()
	go dht.timers()
	return dht.networking.listen()
}

// Bootstrap attempts to bootstrap the network using the BootstrapNodes provided
// to the Options struct. This will trigger an iterativeFindNode to the provided
// BootstrapNodes.
func (dht *DHT) Bootstrap() error {
	if len(dht.options.BootstrapNodes) == 0 {
		return nil
	}
	expectedResponses := []*expectedResponse{}
	wg := &sync.WaitGroup{}

	for _, bn := range dht.options.BootstrapNodes {
		query := &message{}
		query.Sender = dht.ht.Self
		query.Receiver = bn
		query.Type = messageTypePing
		if bn.ID == nil {
			res, err := dht.networking.sendMessage(query, true, -1)
			if err != nil {
				continue
			}
			wg.Add(1)
			expectedResponses = append(expectedResponses, res)
		} else {
			node := newNode(bn)
			dht.addNode(node)
		}
	}

	numExpectedResponses := len(expectedResponses)

	if numExpectedResponses > 0 {
		for _, r := range expectedResponses {
			go func(r *expectedResponse) {
				select {
				case result := <-r.ch:
					// If result is nil, channel was closed
					if result != nil {
						dht.addNode(newNode(result.Sender))
					}
					wg.Done()
					return
				case <-time.After(dht.options.TMsgTimeout):
					dht.networking.cancelResponse(r)
					wg.Done()
					return
				}
			}(r)
		}
	}

	wg.Wait()

	if dht.NumNodes() > 0 {
		_, _, err := dht.iterate(iterateFindNode, dht.ht.Self.ID, nil)
		return err
	}

	return nil
}

// Disconnect will trigger a disconnect from the network. All underlying sockets
// will be closed.
func (dht *DHT) Disconnect() error {
	// TODO if .CreateSocket() is called, but .Listen() is never called, we
	// don't provide a way to close the socket
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

	// According to the Kademlia white paper, after a round of FIND_NODE RPCs
	// fails to provide a node closer than closestNode, we should send a
	// FIND_NODE RPC to all remaining nodes in the shortlist that have not
	// yet been contacted.
	queryRest := false

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

	removeFromShortlist := []*NetworkNode{}

	for {
		expectedResponses := []*expectedResponse{}
		numExpectedResponses := 0

		// Next we send messages to the first (closest) alpha nodes in the
		// shortlist and wait for a response

		for i, node := range sl.Nodes {
			// Contact only alpha nodes
			if i >= alpha && !queryRest {
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
				query.Type = messageTypeFindNode
				queryData := &queryDataFindNode{}
				queryData.Target = target
				query.Data = queryData
			case iterateFindValue:
				query.Type = messageTypeFindValue
				queryData := &queryDataFindValue{}
				queryData.Target = target
				query.Data = queryData
			case iterateStore:
				query.Type = messageTypeFindNode
				queryData := &queryDataFindNode{}
				queryData.Target = target
				query.Data = queryData
			default:
				panic("Unknown iterate type")
			}

			// Send the async queries and wait for a response
			res, err := dht.networking.sendMessage(query, true, -1)
			if err != nil {
				// Node was unreachable for some reason. We will have to remove
				// it from the shortlist, but we will keep it in our routing
				// table in hopes that it might come back online in the future.
				removeFromShortlist = append(removeFromShortlist, query.Receiver)
				continue
			}

			expectedResponses = append(expectedResponses, res)
		}

		for _, n := range removeFromShortlist {
			sl.RemoveNode(n)
		}

		numExpectedResponses = len(expectedResponses)

		resultChan := make(chan (*message))
		for _, r := range expectedResponses {
			go func(r *expectedResponse) {
				select {
				case result := <-r.ch:
					if result == nil {
						// Channel was closed
						return
					}
					dht.addNode(newNode(result.Sender))
					resultChan <- result
					return
				case <-time.After(dht.options.TMsgTimeout):
					dht.networking.cancelResponse(r)
					return
				}
			}(r)
		}

		var results []*message
		if numExpectedResponses > 0 {
		Loop:
			for {
				select {
				case result := <-resultChan:
					if result != nil {
						results = append(results, result)
					} else {
						numExpectedResponses--
					}
					if len(results) == numExpectedResponses {
						close(resultChan)
						break Loop
					}
				case <-time.After(dht.options.TMsgTimeout):
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
					sl.AppendUniqueNetworkNodes(responseData.Closest)
				case iterateFindValue:
					responseData := result.Data.(*responseDataFindValue)
					// TODO When an iterativeFindValue succeeds, the initiator must
					// store the key/value pair at the closest node seen which did
					// not return the value.
					if responseData.Value != nil {
						return responseData.Value, nil, nil
					}
					sl.AppendUniqueNetworkNodes(responseData.Closest)
				case iterateStore:
					responseData := result.Data.(*responseDataFindNode)
					sl.AppendUniqueNetworkNodes(responseData.Closest)
				}
			}
		}

		if !queryRest && len(sl.Nodes) == 0 {
			return nil, nil, nil
		}

		sort.Sort(sl)

		// If closestNode is unchanged then we are done
		if bytes.Compare(sl.Nodes[0].ID, closestNode.ID) == 0 || queryRest {
			// We are done
			switch t {
			case iterateFindNode:
				if !queryRest {
					queryRest = true
					continue
				}
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
					query.Type = messageTypeStore
					queryData := &queryDataStore{}
					queryData.Data = data
					query.Data = queryData
					dht.networking.sendMessage(query, false, -1)
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
		query.Type = messageTypePing
		res, err := dht.networking.sendMessage(query, true, -1)
		if err != nil {
			bucket = append(bucket, node)
			bucket = bucket[1:]
		} else {
			select {
			case <-res.ch:
				return
			case <-time.After(dht.options.TPingMax):
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
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			// Refresh
			for i := 0; i < b; i++ {
				if time.Since(dht.ht.getRefreshTimeForBucket(i)) > dht.options.TRefresh {
					id := dht.ht.getRandomIDFromBucket(k)
					dht.iterate(iterateFindNode, id, nil)
				}
			}

			// Replication
			keys := dht.store.GetAllKeysForReplication()
			for _, key := range keys {
				value, _ := dht.store.Retrieve(key)
				dht.iterate(iterateStore, key, value)
			}

			// Expiration
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
			case messageTypeFindNode:
				data := msg.Data.(*queryDataFindNode)
				dht.addNode(newNode(msg.Sender))
				closest := dht.ht.getClosestContacts(k, data.Target, []*NetworkNode{msg.Sender})
				response := &message{IsResponse: true}
				response.Sender = dht.ht.Self
				response.Receiver = msg.Sender
				response.Type = messageTypeFindNode
				responseData := &responseDataFindNode{}
				responseData.Closest = closest.Nodes
				response.Data = responseData
				dht.networking.sendMessage(response, false, msg.ID)
			case messageTypeFindValue:
				data := msg.Data.(*queryDataFindValue)
				dht.addNode(newNode(msg.Sender))
				value, exists := dht.store.Retrieve(data.Target)
				response := &message{IsResponse: true}
				response.ID = msg.ID
				response.Receiver = msg.Sender
				response.Sender = dht.ht.Self
				response.Type = messageTypeFindValue
				responseData := &responseDataFindValue{}
				if exists {
					responseData.Value = value
				} else {
					closest := dht.ht.getClosestContacts(k, data.Target, []*NetworkNode{msg.Sender})
					responseData.Closest = closest.Nodes
				}
				response.Data = responseData
				dht.networking.sendMessage(response, false, msg.ID)
			case messageTypeStore:
				data := msg.Data.(*queryDataStore)
				dht.addNode(newNode(msg.Sender))
				key := dht.store.GetKey(data.Data)
				expiration := dht.getExpirationTime(key)
				replication := time.Now().Add(dht.options.TReplicate)
				dht.store.Store(key, data.Data, replication, expiration, false)
			case messageTypePing:
				response := &message{IsResponse: true}
				response.Sender = dht.ht.Self
				response.Receiver = msg.Sender
				response.Type = messageTypePing
				dht.networking.sendMessage(response, false, msg.ID)
			}
		case <-dht.networking.getDisconnect():
			dht.networking.messagesFin()
			return
		}
	}
}
