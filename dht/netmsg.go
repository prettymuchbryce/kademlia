package dht

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"net"
)

const (
	messageTypeQueryPing         = "QUERY_PING"
	messageTypeResponsePing      = "RESPONSE_PING"
	messageTypeQueryStore        = "QUERY_STORE"
	messageTypeResponseStore     = "RESPONSE_STORE"
	messageTypeQueryFindNode     = "QUERY_FIND_NODE"
	messageTypeResponseFindNode  = "RESPONSE_FIND_NODE"
	messageTypeQueryFindValue    = "QUERY_FIND_VALUE"
	messageTypeResponseFindValue = "RESPONSE_FIND_VALUE"
)

type message struct {
	Node  *node
	ID    [b]byte
	Error error
	Type  string
	Data  interface{}
}

type queryDataFindNode struct {
	Target []byte
}

type queryDataFindValue struct {
	Key []byte
}

type queryDataStore struct {
	Data []byte
	Key  []byte
}

type responseDataFindNode struct {
	Closest []*node
}

type responseDataFindValue struct {
	Closest []*node
	Value   []byte
}

type responseDataStore struct {
	Success bool
}

func serializeMessage(q *message) ([]byte, error) {
	var msgBuffer bytes.Buffer
	enc := gob.NewEncoder(&msgBuffer)
	err := enc.Encode(q)
	if err != nil {
		return nil, err
	}

	length := msgBuffer.Len()

	var lengthBytes []byte
	binary.PutUvarint(lengthBytes, uint64(length))

	var result []byte
	result = append(result, lengthBytes...)
	result = append(result, msgBuffer.Bytes()...)

	return result, nil
}

func deserializeMessage(conn net.Conn) (*message, error) {
	lengthBytes := make([]byte, 4)
	_, err := conn.Read(lengthBytes)
	if err != nil {
		return nil, err
	}

	lengthReader := bytes.NewBuffer(lengthBytes)
	length, err := binary.ReadUvarint(lengthReader)
	if err != nil {
		return nil, err
	}

	msgBytes := make([]byte, length)
	_, err = conn.Read(msgBytes)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewBuffer(msgBytes)
	msg := &message{}
	dec := gob.NewDecoder(reader)

	err = dec.Decode(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
