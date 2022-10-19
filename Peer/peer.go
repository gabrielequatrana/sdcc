package main

import (
	"encoding/json"
	"fmt"
	"github.com/phayes/freeport"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"prog/Utils"
	"strconv"
	"time"
)

type api int

var ID int
var peerList []Utils.Peer
var conf Utils.Conf
var ip, port string

var election bool
var coordinator int
var ch chan int
var hbch chan int

func main() {

	fmt.Println("Peer service startup")

	// Make GO channels
	ch = make(chan int)
	hbch = make(chan int)

	// Reading config file to retrieve IP address and port
	fmt.Println("Reading config file")
	j, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalln("Open error: ", err)
	}

	// Unmarshalling json file
	err = json.Unmarshal(j, &conf)
	if err != nil {
		log.Fatalln("Unmarshal error: ", err)
	}
	fmt.Println("Conf: ", conf)

	// Registering RPC API
	err = rpc.RegisterName("Peer", new(api))
	rpc.HandleHTTP()

	// Connect to register service
	regIP := conf.Register.IP
	regPort := conf.Register.Port
	addr := regIP + ":" + regPort
	cli, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatalln("Error dial: ", err)
	}

	// Set random port
	p, err := freeport.GetFreePort()
	if err != nil {
		log.Fatalln("Error GetFreePort: ", err)
	}

	// Open connection
	ip = conf.Peer.IP
	port = strconv.Itoa(p)
	lis, err := net.Listen("tcp", ip+":"+port)
	if err != nil {
		log.Fatalln("Listen error: ", err)
	}

	// Initialize struct peer
	peer := Utils.Peer{
		IP:   ip,
		Port: port,
	}

	// Call remote method RegisterPeer
	var reply Utils.RegistrationReply
	err = cli.Call("Register.RegisterPeer", &peer, &reply)
	if err != nil {
		log.Fatalln("Error call: ", err)
	}

	// Setting peer ID and retrieve information about other peers
	ID = reply.ID
	peerList = reply.Peers
	fmt.Println("Resp: {id : ", ID, "}, {lis : ", peerList, "}")
	err = cli.Close()
	if err != nil {
		log.Fatalln("Error close: ", err)
	}

	// Goroutine for serve RPC request coming from other peers
	go func() {
		err := http.Serve(lis, nil)
		if err != nil {
			log.Fatalln("Error serve: ", err)
		}
	}()

	// Goroutine for HeartBeat monitoring
	go heartbeat()

	// TODO: Initially all Peer know the coordinator (fare in modo che non sia cosi)
	if ID == peerList[len(peerList)-1].ID {
		sendCoordinator()
	}

	// Infinite loop executed by peer.
	// Wait for message received in channel and call functions.
	for {
		select {
		case msg1 := <-ch:

			switch msg1 {

			// If the peer receive an ELECTION message he has to create a new election
			case Utils.ELECTION:
				sendElection()
				if election {
					sendCoordinator()
				}
			}

		case msg2 := <-hbch:
			fmt.Println("Peer \"", ID, "\" know that peer", msg2, "is down.")
		}
	}
}

// SendMessage RPC method provided by peers
func (t *api) SendMessage(args *Utils.Message, reply *Utils.Message) error {

	// Message sent by a peer
	msg := args.Msg

	// Check type of message received
	switch msg {

	// ELECTION message
	case Utils.ELECTION:
		fmt.Println("Peer \"", ID, "\" received: ELECTION from:", args.ID)

		// If the current peer has a greater id, send OK message.
		if ID > args.ID {
			reply.Msg = Utils.OK
			ch <- msg // Send message to channel
		}

		// TODO cosa fa altrimenti?
		// TODO (in realta credo non serva perche gli ELECTION vengono inviati solo ai peer con id minori)

	// COORDINATOR message
	case Utils.COORDINATOR:
		fmt.Println("Peer \"", ID, "\" recognized as coordinator", args.ID)

		// Set coordinator ID
		coordinator = args.ID

	// HEARTBEAT message
	case Utils.HEARTBEAT:
		fmt.Println("Peer \"", ID, "\" received: HEARTBEAT from:", args.ID)

		// Set reply msg parameters
		reply.ID = ID
		reply.Msg = Utils.HEARTBEAT
	}

	// No error to manage
	return nil
}

// Send ELECTION message to all peers with greater id
func sendElection() {
	var reply Utils.Message // Reply message
	election = true         // The current peer take part in the election

	// Send ELECTION to peers
	for _, p := range peerList {
		if p.ID > ID {
			fmt.Println("Peer \"", ID, "\" sending initial election to: ", p.ID)

			// Send message to p
			err := send(ID, Utils.ELECTION, p, &reply)
			if err != nil {
				log.Fatalln("Error call: ", err)
			}

			fmt.Println("Peer \"", ID, "\" received from", p.ID, ": ", reply.Msg)

			// If the current peer receive an OK message, it exits the election
			if reply.Msg == Utils.OK && election {
				election = false
				fmt.Println("Peer \"", ID, "\" exit the election")
			}
		}
	}
}

// Send COORDINATOR message to all peers
func sendCoordinator() {
	var reply Utils.Message // Reply message

	// Set coordinator as peer id
	coordinator = ID
	fmt.Println("Peer \"", ID, "\" recognized as coordinator himself")

	// Send COORDINATOR to peers
	for _, p := range peerList {
		if p.ID != ID {
			fmt.Println("Peer \"", ID, "\" sending COORDINATOR to: ", p.ID)

			// Send message to p
			err := send(ID, Utils.COORDINATOR, p, &reply)
			if err != nil {
				log.Fatalln("Error call: ", err)
			}
		}
	}
}

// Check peers status by sending heartbeat message
func heartbeat() {

	// Execute an infinite loop
	for {
		time.Sleep(time.Second * 30) // TODO Repeat every two second (creare parametro)

		// Send heartbeat to all peers
		for _, p := range peerList {
			beat := new(Utils.Message)
			if p.ID != ID {
				fmt.Println("Peer \"", ID, "\" sending heartbeat to: ", p.ID)

				// Send heartbeat to p, if p crashed send ERROR to heartbeat channel
				err := send(ID, Utils.HEARTBEAT, p, beat)
				if err != nil {
					fmt.Println("Peer \"", ID, "\" BEAT NOT RECEIVED from: ", p.ID)
					hbch <- p.ID
				}

				// If the peer responds than it is alive
				if beat.Msg == Utils.HEARTBEAT {
					fmt.Println("Peer \"", ID, "\" says", beat.ID, "is alive")
				}
			}
		}
	}
}

// Send a message to a specific peer
func send(id int, msg int, peer Utils.Peer, reply *Utils.Message) error {

	// Make a new message to send
	message := Utils.Message{
		ID:  id,
		Msg: msg,
	}

	// Connect to the receiver peer
	cli, err := rpc.DialHTTP("tcp", peer.IP+":"+peer.Port)
	if err != nil {
		return err
	}

	// Call the RPC method SendMessage exposed by the receiver peer
	err = cli.Call("Peer.SendMessage", &message, &reply)
	if err != nil {
		return err
	}

	// Return nil if there's no error
	return nil
}
