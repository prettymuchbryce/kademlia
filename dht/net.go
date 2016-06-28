package dht

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

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

var errorDisconnected = errors.New("disconnected")

// Network TODO
type networking interface {
	sendMessages([]*message, bool) ([]*message, error)
	getMessage() (*message, error)
	init()
	createSocket(host string, port string) error
	listen() error
	disconnect() error
}

type realNetworking struct {
	socket     *utp.Socket
	sendChan   chan ([]*message)
	dcChan     chan (int)
	address    *net.UDPAddr
	broadcast  *broadcast
	connection *net.UDPConn
	waitgroup  *sync.WaitGroup
	mutex      *sync.Mutex
}

func (rn *realNetworking) getMessage() (*message, error) {
	rn.waitgroup.Add(1)
	defer rn.waitgroup.Done()
	id, c := rn.broadcast.addListener("recv")
	for {
		msg := <-c
		if msg == nil {
			// If message is nil channel has been closed and we have been
			// disconnected.
			rn.broadcast.removeListener(id, "recv")
			return nil, errorDisconnected
		}
		rn.broadcast.removeListener(id, "recv")
		return msg.(*message), nil
	}
}

func (rn *realNetworking) sendMessages(msgs []*message, expectResponse bool) ([]*message, error) {
	rn.waitgroup.Add(1)
	defer rn.waitgroup.Done()
	var id int
	var c chan (interface{})

	if expectResponse {
		id, c = rn.broadcast.addListener("recv")
	}
	rn.sendChan <- msgs
	if expectResponse {
		var result []*message
		for {
			msg := <-c
			if msg == nil {
				// If message is nil channel has been closed and we have been
				// disconnected.
				rn.broadcast.removeListener(id, "recv")
				return nil, errorDisconnected
			}
			for _, m := range msgs {
				if m.ID == msg.(*message).ID {
					result = append(result, msg.(*message))
					if len(result) == len(msgs) {
						rn.broadcast.removeListener(id, "recv")
						return result, nil
					}
				}
			}
		}
	} else {
		return nil, nil
	}
}

func (rn *realNetworking) init() {
	if rn.mutex == nil {
		rn.mutex = &sync.Mutex{}
	}
	rn.mutex.Lock()
	defer rn.mutex.Unlock()
	rn.waitgroup = &sync.WaitGroup{}
	netMsgInit()
	rn.sendChan = make(chan ([]*message))
	rn.dcChan = make(chan (int))
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
	rn.mutex.Lock()
	defer rn.mutex.Unlock()
	if rn.socket == nil {
		panic(errors.New("Can't send. Not initialized."))
	}
	conn, err := rn.socket.Dial(msg.Receiver.IP.String() + ":" + strconv.Itoa(msg.Receiver.Port))
	if err != nil {
		return err
	}

	rn.waitgroup.Add(1)

	data, err := serializeMessage(msg)
	if err != nil {
		return err
	}

	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	defer rn.waitgroup.Done()
	conn.Close()
	conn = nil

	return nil
}

func (rn *realNetworking) disconnect() error {
	rn.mutex.Lock()
	defer rn.mutex.Unlock()
	close(rn.dcChan)
	close(rn.sendChan)
	err := rn.socket.Close()
	rn.socket = nil
	rn.broadcast.removeAll()
	rn.waitgroup.Wait()
	rn.sendChan = make(chan ([]*message))
	rn.dcChan = make(chan (int))
	return err
}

func (rn *realNetworking) listen() error {
	rn.waitgroup.Add(1)
	defer rn.waitgroup.Done()

	if rn.socket == nil {
		// Possible that we were disconnected before we could start listening
		panic(errors.New("No socket"))
		return nil
	}

	go func() {
		for {
			rn.waitgroup.Add(1)
			defer rn.waitgroup.Done()
			msgs := <-rn.sendChan
			if msgs == nil {
				// Channel closed, disconnected
				return
			}
			for _, msg := range msgs {
				rn.send(msg)
			}
		}
	}()

	for {
		conn, err := rn.socket.Accept()
		if err == nil {
			rn.waitgroup.Add(1)
		}

		if err != nil {
			if err.Error() == "closed" {
				conn = nil
				return nil
			} else {
				return err
			}
		}

		go func(conn net.Conn) {
			for {
				// Wait for messages
				msg, err := deserializeMessage(conn)
				if err != nil {
					if err == io.EOF {
						defer rn.waitgroup.Done()
						conn.Close()
						conn = nil
						return
					} else {
						fmt.Println(err)
						continue
					}
				}
				rn.broadcast.dispatch("recv", msg)
			}
		}(conn)

		go func(conn net.Conn) {
			rn.waitgroup.Add(1)
			defer rn.waitgroup.Done()
			_ = <-rn.dcChan
			conn.Close()
			conn = nil
		}(conn)

		conn = nil

	}
	return nil
}
