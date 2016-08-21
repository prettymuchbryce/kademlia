package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/prettymuchbryce/kademlia"

	"gopkg.in/readline.v1"
)

func main() {
	var ip = flag.String("ip", "127.0.0.1", "IP Address to use")
	var port = flag.String("port", "", "Port to use")
	var bIP = flag.String("bip", "127.0.0.1", "IP Address to bootstrap against")
	var bPort = flag.String("bport", "", "Port to bootstrap against")
	var help = flag.Bool("help", false, "Display Help")

	flag.Parse()

	if *help {
		displayFlagHelp()
		os.Exit(0)
	}

	if *ip == "" {
		displayFlagHelp()
		os.Exit(0)
	}

	if *port == "" {
		displayFlagHelp()
		os.Exit(0)
	}

	var bootstrapNodes []*kademlia.NetworkNode
	if *bIP != "" || *bPort != "" {
		bootstrapNode := kademlia.NewNetworkNode(*bIP, *bPort)
		bootstrapNodes = append(bootstrapNodes, bootstrapNode)
	}

	dht, err := kademlia.NewDHT(&kademlia.MemoryStore{}, &kademlia.Options{
		BootstrapNodes: bootstrapNodes,
		IP:             *ip,
		Port:           *port,
	})

	fmt.Println("Opening socket..")
	err = dht.CreateSocket()
	if err != nil {
		panic(err)
	}
	fmt.Println("..done")

	go func() {
		fmt.Sprintln("Now listening on port %s", port)
		err := dht.Listen()
		panic(err)
	}()

	if len(bootstrapNodes) > 0 {
		fmt.Println("Bootstrapping..")
		dht.Bootstrap()
		fmt.Println("..done")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			err := dht.Disconnect()
			if err != nil {
				panic(err)
			}
		}
	}()

	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			break
		}
		input := strings.Split(line, " ")
		switch input[0] {
		case "help":
			displayHelp()
		case "store":
			if len(input) != 2 {
				displayHelp()
				continue
			}
			id, err := dht.Store([]byte(input[1]))
			if err != nil {
				println(err.Error())
			}
			println("Stored ID: " + id)
		case "get":
			if len(input) != 2 {
				displayHelp()
				continue
			}
			data, exists, err := dht.Get(input[1])
			if err != nil {
				println(err.Error())
			}
			println("Searching for", input[1])
			if exists {
				println("..Found data:", string(data))
			} else {
				println("..Nothing found for this key!")
			}
		case "info":
			nodes := dht.NumNodes()
			self := dht.GetSelfID()
			println("IP: " + *ip)
			println("Port: " + *port)
			println("ID: " + self)
			println("Known Nodes: " + strconv.Itoa(nodes))
		}
	}
}

func displayFlagHelp() {
	println(`cli-example

Usage:
	cli-example --port [port]

Options:
	--help Show this screen.
	--ip=<ip> Local IP [default: 127.0.0.1]
	--port=[port] Local Port
	--bip=<ip> Bootstrap IP [default: 127.0.0.1]
	--bport<port> Bootstrap Port`)
}

func displayHelp() {
	println(`
help - This message
store <message> - Store a message on the network
get <key> - Get a message from the network
info - Display information about this node
	`)
}
