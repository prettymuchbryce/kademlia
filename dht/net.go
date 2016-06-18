package dht

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/anacrolix/utp"
)

// NetworkNode is the network's representation of a node
type NetworkNode struct {
	// ID is a 20 byte unique identifier
	ID []byte

	// IP is the IPv4 address of the node
	IP net.IP

	// Port is the port of the node
	Port int
}

// Network TODO
type networking interface {
	sendMessages([]*message) []*message
	getMessage() *message
	init()
	createSocket(host string, port string) error
	listen() error
}

type realNetworking struct {
	socket     *utp.Socket
	recvChan   chan (*message)
	sendChan   chan ([]*message)
	address    *net.UDPAddr
	broadcast  *broadcast
	connection *net.UDPConn
}

func newNetworking() *realNetworking {
	net := &realNetworking{}
	net.recvChan = make(chan (*message))
	net.sendChan = make(chan ([]*message))
	return net
}

func (rn *realNetworking) getMessage() *message {
	id, c := rn.broadcast.addListener("recv")
	for {
		msg := <-c
		rn.broadcast.removeListener(id, "recv")
		return msg.(*message)
	}
}

func (rn *realNetworking) sendMessages(msgs []*message) []*message {
	id, c := rn.broadcast.addListener("recv")
	rn.sendChan <- msgs
	var result []*message
	for {
		msg := <-c
		for _, m := range msgs {
			if m.ID == msg.(*message).ID {
				result = append(result, msg.(*message))
				if len(result) == len(msgs) {
					rn.broadcast.removeListener(id, "recv")
					return result
				}
			}
		}
	}
}

func (rn *realNetworking) init() {
	netMsgInit()
	rn.recvChan = make(chan (*message))
	rn.sendChan = make(chan ([]*message))
	rn.broadcast = &broadcast{}
	rn.broadcast.init()
}

func (rn *realNetworking) createSocket(host string, port string) error {
	socket, err := utp.NewSocket("udp", host+":"+port)
	if err != nil {
		return err
	}

	rn.socket = socket

	return nil
}

func (rn *realNetworking) send(msg *message) error {
	if rn.socket == nil {
		panic(errors.New("Not initialized."))
	}
	conn, err := rn.socket.Dial(msg.Receiver.IP.String() + ":" + strconv.Itoa(msg.Receiver.Port))
	if err != nil {
		return err
	}

	data, err := serializeMessage(msg)
	if err != nil {
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (rn *realNetworking) listen() error {
	if rn.socket == nil {
		panic(errors.New("Not initialized."))
	}

	go func() {
		for {
			msgs := <-rn.sendChan
			for _, msg := range msgs {
				rn.send(msg)
			}
		}
	}()

	for {
		conn, err := rn.socket.Accept()
		if err != nil {
			fmt.Println(err)
		}

		go func() {
			for {
				// Wait for messages
				msg, err := deserializeMessage(conn)
				if err != nil {
					fmt.Println(err)
					continue
				}
				rn.broadcast.dispatch("recv", msg)
			}
		}()
	}

	return nil
}
