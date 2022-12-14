package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"prog/Utils"
	"strconv"

	"github.com/joho/godotenv"
)

type RegisterApi int // Used to publish RPC method

var numPeer int           // Number of peers in the network
var currentPeer = 0       // ID of the current peer served
var peerList []Utils.Peer // List of peers in the network
var conf Utils.Conf       // Configuration of peer and register service

var ch chan int // Go channel to wait for all peers to complete registration

func main() {

	log.Println("Register service startup, reading config and env files.")

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Load env file error:", err)
	}

	// Setting number of peers
	numPeer, err = strconv.Atoi(os.Getenv("PEERS"))
	if err != nil {
		log.Fatalln("AtoI peers number error:", err)
	}

	// Make GO channel
	ch = make(chan int)

	// Reading config file to retrieve IP address and port
	j, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalln("Open config file error:", err)
	}

	// Unmarshalling json file
	err = json.Unmarshal(j, &conf)
	if err != nil {
		log.Fatalln("Unmarshal config file error:", err)
	}

	// Registering the RPC method
	err = rpc.RegisterName("Register", new(RegisterApi))
	if err != nil {
		log.Fatalln("RegisterName error:", err)
	}

	// Handle HTTP request
	rpc.HandleHTTP()

	// Register service listening to incoming request
	lis, err := net.Listen("tcp", ":"+conf.Register.Port)
	if err != nil {
		log.Fatalln("Listen error:", err)
	}

	// Goroutine that wait all peer before the register service sends the reply
	go func() {
		for currentPeer < numPeer {
			// Wait
		}
		log.Println("Register service built this list:", peerList)
		for i := 0; i <= numPeer; i++ {
			ch <- 1 // Send message to ch to resume the execution of RegisterPeer
		}
	}()

	// Serve incoming request
	err = http.Serve(lis, nil)
	if err != nil {
		log.Fatalln("Serve error:", err)
	}
}

// RegisterPeer Exported method that peers call to register on the network
func (t *RegisterApi) RegisterPeer(args *Utils.Peer, reply *Utils.RegistrationReply) error {

	// Retrieve peer port
	port, err := strconv.Atoi(args.Port)
	if err != nil {
		log.Fatalln("AtoI peer port error:", err)
	}

	// Create Peer struct to add to list
	peer := Utils.Peer{
		ID:   currentPeer,
		IP:   args.IP,
		Port: strconv.Itoa(port),
	}

	// Add registered peer to the list
	peerList = append(peerList, peer)

	// Add to the reply the peer ID
	reply.ID = currentPeer

	// Increment currentPeer
	currentPeer++

	// Wait all peers before sends reply
	<-ch

	// Add to the reply the peer list
	reply.Peers = peerList

	return nil
}
