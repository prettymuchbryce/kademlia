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

var (
	errorValueNotFound = errors.New("Value not found")
)

// Network TODO
type networking interface {
	sendMessage(*message, bool) (chan (*message), error)
	getMessage() *message
	getMessageFin()
	init()
	createSocket(host string, port string) error
	listen() error
	disconnect() error
}

type realNetworking struct {
	socket      *utp.Socket
	sendChan    chan (*message)
	recvChan    chan (*message)
	dcChan      chan (int)
	address     *net.UDPAddr
	connection  *net.UDPConn
	mutex       *sync.Mutex
	msgCounter  int64
	responseMap map[int64]chan (*message)
}

func (rn *realNetworking) init() {
	rn.mutex = &sync.Mutex{}
	rn.sendChan = make(chan (*message))
	rn.recvChan = make(chan (*message))
	rn.dcChan = make(chan (int))
	rn.responseMap = make(map[int64]chan (*message))
}

func (rn *realNetworking) getMessage() *message {
	return <-rn.recvChan
}

func (rn *realNetworking) getMessageFin() {
	rn.dcChan <- 1
}

func (rn *realNetworking) createSocket(host string, port string) error {
	socket, err := utp.NewSocket("udp", host+":"+port)
	if err != nil {
		return err
	}

	rn.socket = socket

	return nil
}

func (rn *realNetworking) sendMessage(msg *message, expectResponse bool) (chan (*message), error) {
	rn.mutex.Lock()
	defer rn.mutex.Unlock()
	id := rn.msgCounter
	rn.msgCounter++

	conn, err := rn.socket.Dial(msg.Receiver.IP.String() + ":" + strconv.Itoa(msg.Receiver.Port))
	if err != nil {
		return nil, err
	}

	data, err := serializeMessage(msg)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	if expectResponse {
		responseChan := make(chan (*message))
		rn.responseMap[id] = responseChan
		return responseChan, nil
	}

	conn.Close()
	return nil, nil
}

func (rn *realNetworking) disconnect() error {
	close(rn.sendChan)
	close(rn.recvChan)
	<-rn.dcChan
	err := rn.socket.Close()
	return err
}

func (rn *realNetworking) listen() error {
	for {
		conn, err := rn.socket.Accept()
		if err != nil {
			return err
		}

		go func(conn net.Conn) {
			for {
				// Wait for messages
				msg, err := deserializeMessage(conn)
				if err != nil {
					if err == io.EOF {
						conn.Close()
						conn = nil
						return
					} else {
						fmt.Println(err)
						continue
					}
				}

				rn.mutex.Lock()
				if msg.IsResponse && rn.responseMap[msg.ID] != nil {
					rn.responseMap[msg.ID] <- msg
					close(rn.responseMap[msg.ID])
					delete(rn.responseMap, msg.ID)
				} else {
					rn.recvChan <- msg
				}
				rn.mutex.Unlock()
			}
		}(conn)
	}

	return nil
}
