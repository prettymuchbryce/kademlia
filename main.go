package main

import "kademlia/dht"

func main() {
	dht, err := dht.NewDHT("", nil)
	if err != nil {
		panic(err)
	}
	dht.Connect()
	dht.Run()
}
