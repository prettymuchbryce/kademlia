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
	var ip = flag.String("ip", "0.0.0.0", "IP Address to use")
	var port = flag.String("port", "0", "Port to use")
	var bIP = flag.String("bip", "", "IP Address to bootstrap against")
	var bPort = flag.String("bport", "", "Port to bootstrap against")
	var help = flag.Bool("help", false, "Display Help")
	var stun = flag.Bool("stun", true, "Use STUN")

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
		UseStun:        *stun,
	})

	fmt.Println("Opening socket..")

	if *stun {
		fmt.Println("Discovering public address using STUN..")
	}

	err = dht.CreateSocket()
	if err != nil {
		panic(err)
	}
	fmt.Println("..done")

	go func() {
		fmt.Println("Now listening on " + dht.GetNetworkAddr())
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
		for range c {
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
				fmt.Println(err.Error())
			}
			fmt.Println("Stored ID: " + id)
		case "get":
			if len(input) != 2 {
				displayHelp()
				continue
			}
			data, exists, err := dht.Get(input[1])
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println("Searching for", input[1])
			if exists {
				fmt.Println("..Found data:", string(data))
			} else {
				fmt.Println("..Nothing found for this key!")
			}
		case "info":
			nodes := dht.NumNodes()
			self := dht.GetSelfID()
			addr := dht.GetNetworkAddr()
			fmt.Println("Addr: " + addr)
			fmt.Println("ID: " + self)
			fmt.Println("Known Nodes: " + strconv.Itoa(nodes))
		}
	}
}

func displayFlagHelp() {
	fmt.Println(`cli-example

Usage:
	cli-example --port [port]

Options:
	--help Show this screen.
	--ip=<ip> Local IP [default: 0.0.0.0]
	--port=[port] Local Port [default: 0]
	--bip=<ip> Bootstrap IP
	--bport=<port> Bootstrap Port
	--stun=<bool> Use STUN protocol for public addr discovery [default: true]`)
}

func displayHelp() {
	fmt.Println(`
help - This message
store <message> - Store a message on the network
get <key> - Get a message from the network
info - Display information about this node
	`)
}
