package main

import (
	"errors"
	"net"
)

type mockNetworking struct {
	recv     chan (*message)
	send     chan (*message)
	failNext bool
}

func newMockNetworking() *mockNetworking {
	net := &mockNetworking{}
	return net
}

func (net *mockNetworking) listen() error {
	return nil
}

func (net *mockNetworking) disconnect() error {
	close(net.recv)
	close(net.send)
	return nil
}

func (net *mockNetworking) createSocket(host string, port string) error {
	return nil
}

func (net *mockNetworking) cancelResponse(*expectedResponse) {
}

func (net *mockNetworking) init(self *NetworkNode) {
	net.recv = make(chan (*message))
	net.send = make(chan (*message))
}

func (net *mockNetworking) messagesFin() {

}

func (net *mockNetworking) timersFin() {

}

func (net *mockNetworking) getDisconnect() chan (int) {
	return nil
}

func (net *mockNetworking) getMessage() chan (*message) {
	return nil
}

func (net *mockNetworking) failNextSendMessage() {
	net.failNext = true
}

func (net *mockNetworking) sendMessage(q *message, id int64, expectResponse bool) (*expectedResponse, error) {
	if net.failNext {
		net.failNext = false
		return nil, errors.New("MockNetworking Error")
	}
	net.recv <- q
	if expectResponse {
		return &expectedResponse{ch: net.send, query: q, node: q.Receiver, id: id}, nil
	} else {
		return nil, nil
	}
}

func mockFindNodeResponse(query *message, nextID []byte) *message {
	r := &message{}
	n := newNode(&NetworkNode{})
	n.ID = query.Sender.ID
	n.IP = query.Sender.IP
	n.Port = query.Sender.Port
	r.Receiver = n.NetworkNode
	r.Sender = &NetworkNode{ID: query.Receiver.ID, IP: net.ParseIP("0.0.0.0"), Port: 3001}
	r.Type = query.Type
	r.IsResponse = true
	responseData := &responseDataFindNode{}
	responseData.Closest = []*NetworkNode{&NetworkNode{IP: net.ParseIP("0.0.0.0"), Port: 3001, ID: nextID}}
	r.Data = responseData
	return r
}
