package dht

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"sort"

	b58 "github.com/jbenet/go-base58"
)

// Public interface

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

// Store TODO
func (dht *DHT) Store(data []byte) (string, error) {
	sha := sha1.New()
	key := sha.Sum(data)
	dht.store.Store(key, data)
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
		ip = "127.0.0.1"
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
	return dht.networking.listen()
}

func (dht *DHT) Bootstrap() error {
	if len(dht.options.BootstrapNodes) > 0 {
		for _, bn := range dht.options.BootstrapNodes {
			node := newNode(bn)
			dht.ht.addNode(node)
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

			queries = append(queries, query)
		}

		// Send the async queries and wait for a response
		var chans []chan (*message)
		for _, q := range queries {
			ch, err := dht.networking.sendMessage(q, dht.msgCounter, true)
			dht.msgCounter++
			if err != nil {
				// TODO handle this somehow ?
				fmt.Println(err)
				continue

			}
			chans = append(chans, ch)
		}

		var results []*message
		for _, c := range chans {
			result := <-c
			// TODO handle errors/timeouts
			results = append(results, result)
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
					dht.ht.addNode(newNode(n))
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
					dht.ht.addNode(newNode(n))
				}
				sl.AppendUniqueNetworkNodes(responseData.Closest)
			case iterateStore:
				responseData := result.Data.(*responseDataFindNode)
				for _, n := range responseData.Closest {
					dht.ht.addNode(newNode(n))
				}
				sl.AppendUniqueNetworkNodes(responseData.Closest)
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

func (dht *DHT) listen() {
	for {
		msg := dht.networking.getMessage()
		if msg == nil {
			// Disconnected
			dht.networking.getMessageFin()
			return
		}
		switch msg.Type {
		case messageTypeQueryFindNode:
			data := msg.Data.(*queryDataFindNode)
			dht.ht.addNode(newNode(msg.Sender))
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
			dht.ht.addNode(newNode(msg.Sender))
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
			dht.ht.addNode(newNode(msg.Sender))
			dht.store.Store(data.Key, data.Data)
		case messageTypeQueryPing:
			// response := &message{IsResponse: true}
		}
	}
}
