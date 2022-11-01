package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"prog/Utils"
	"strconv"
	"time"
)

type RegisterApi int

var peerList []Utils.Peer
var currentPeer = 0
var numPeer int
var conf Utils.Conf
var verbose = false

func main() {

	fmt.Println("Register service startup")

	// Reading config file to retrieve IP address and port
	fmt.Println("Reading config file")
	j, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalln("Open error: ", err)
	}

	// Load .env file
	err = godotenv.Load()
	if err != nil {
		log.Fatalln("Load env error: ", err)
	}

	// Setting number of peers
	numPeer, err = strconv.Atoi(os.Getenv("PEERS"))
	if err != nil {
		log.Fatalln("Atoi error: ", err)
	}

	// Setting verbose flag
	if os.Getenv("VERBOSE") == "1" {
		verbose = true
	}

	// Unmarshalling json file
	err = json.Unmarshal(j, &conf)
	if err != nil {
		log.Fatalln("Unmarshal error: ", err)
	}
	fmt.Println("Conf: ", conf)

	// Registering the RPC API to export
	err = rpc.RegisterName("Register", new(RegisterApi))
	if err != nil {
		log.Fatalln("RegisterName error: ", err)
	}

	// Handle HTTP request
	rpc.HandleHTTP()

	// Register service listening to incoming request
	lis, err := net.Listen("tcp", ":"+conf.Register.Port)
	if err != nil {
		log.Fatalln("Listen error: ", err)
	}

	// Serve incoming request
	err = http.Serve(lis, nil)
	if err != nil {
		log.Fatalln("Serve error: ", err)
	}
}

// RegisterPeer Exported API to register peer in the network
func (t *RegisterApi) RegisterPeer(args *Utils.Peer, reply *Utils.RegistrationReply) error {

	// Retrieve peer port
	port, err := strconv.Atoi(args.Port)
	if err != nil {
		log.Fatalln("Atoi error: ", err)
	}

	// Create Peer struct to send
	peer := Utils.Peer{
		ID:   currentPeer,
		IP:   args.IP,
		Port: strconv.Itoa(port),
	}

	// Add registered peer to the list
	peerList = append(peerList, peer)
	fmt.Println(peerList)

	// Fill the reply with peer ID and peer list
	reply.ID = currentPeer

	// Increment currentPeer
	currentPeer++

	// While number of peer registered less than initialized peer do nothing
	for currentPeer < numPeer {
		time.Sleep(time.Microsecond) // TODO si può fare senza sleep?
	}

	// Fill the reply with peer ID and peer list
	reply.Peers = peerList

	return nil
}
