package kademlia

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeNetMsg(t *testing.T) {
	netMsgInit()
	var conn bytes.Buffer

	node := newNode(&NetworkNode{})
	id, _ := newID()
	node.ID = id
	node.Port = 3000
	node.IP = net.ParseIP("0.0.0.0")

	msg := &message{}
	msg.Type = messageTypeFindNode
	msg.Receiver = node.NetworkNode
	msg.Data = &queryDataFindNode{
		Target: id,
	}

	serialized, err := serializeMessage(msg)
	if err != nil {
		panic(err)
	}

	conn.Write(serialized)

	deserialized, err := deserializeMessage(&conn)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, msg, deserialized)
}
