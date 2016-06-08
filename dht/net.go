package dht

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/anacrolix/utp"
)

// Network TODO
type networking interface {
	sendMessages([]*message) chan ([]*message)
	getMessage() *message
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

func (rn *realNetworking) init(host string, port string) error {
	socket, err := utp.NewSocket("udp", host+":"+port)
	if err != nil {
		return err
	}

	rn.broadcast.init()

	rn.socket = socket

	return nil
}

func (rn *realNetworking) send(msg *message) error {
	if rn.socket == nil {
		panic(errors.New("Not initialized."))
	}
	conn, err := rn.socket.Dial(msg.Node.IP.String() + ":" + strconv.Itoa(msg.Node.Port))
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
	for {
		conn, err := rn.socket.Accept()
		if err != nil {
			fmt.Println(err)
		}

		go func() {
			for {
				msg := <-rn.sendChan
				rn.sendMessages(msg)
			}
		}()

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
