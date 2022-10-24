package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/phayes/freeport"
	"log"
	"math/rand"
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
var algo string
var delay int
var tries int
var hbtime int
var ch chan int
var hbch chan int

// TODO Aggiungere logica altro algoritmo
// TODO Sistemare bug (Conosciuti: ogni tanto non cancella le immagini docker alla fine,
// TODO vedere se è possibile stampare in ordine, ...)
// TODO Aggiungere VERBOSE

func main() {

	fmt.Println("Peer service startup")

	// Set randomizer seed
	rand.Seed(time.Now().UnixNano())

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Load env error: ", err)
	}

	// Setting tries
	tries, err = strconv.Atoi(os.Getenv("TRIES"))
	if err != nil {
		log.Fatalln("Atoi error: ", err)
	}

	// Setting delay
	delay, err = strconv.Atoi(os.Getenv("DELAY"))
	if err != nil {
		log.Fatalln("Atoi error: ", err)
	}

	// Setting algorithm type
	var a Algorithm
	algo = os.Getenv("ALGO")
	if algo == "bully" {
		a = Bully{}
	} else {
		a = Ring{}
	}

	// Setting heartbeat time
	hbtime, err = strconv.Atoi(os.Getenv("HEARTBEAT"))
	if err != nil {
		log.Fatalln("Atoi error: ", err)
	}

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
		err = http.Serve(lis, nil)
		if err != nil {
			log.Fatalln("Error serve: ", err)
		}
	}()

	// Goroutine for HeartBeat monitoring
	go heartbeat()

	// Initially only the Peer with smaller id sends the ELECTION message
	if ID == peerList[0].ID {
		newElection(a)
	}

	// Infinite loop executed by peer.
	// Wait for message received in channel and call functions.
	for {
		select {
		case <-ch:
			// If the peer receive an ELECTION message he has to create a new election
			newElection(a)

		// Peer msg2 down
		case id := <-hbch:
			fmt.Println("Peer \"", ID, "\" know that peer", id, "is down.")

			// If the peer down is the coordinator make a new election
			if id == coordinator && !election {
				newElection(a)
			}
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
		reply.Msg = Utils.OK
		ch <- msg // Send message to channel

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

	// Random delay in ms
	d := rand.Intn(delay * 1000)
	fmt.Println("Peer \"", ID, "\" generated this delay in ms:", d)
	time.Sleep(time.Duration(d) * time.Millisecond)

	// No error to manage
	return nil
}

func newElection(alg Algorithm) {
	alg.sendElection()
	if election {
		alg.sendCoordinator()
	}
}

// Send ELECTION message to all peers with greater id
//func sendElection() {
//	var reply Utils.Message // Reply message
//	election = true         // The current peer take part in the election
//
//	// Send ELECTION to peers
//	for _, p := range peerList {
//		if p.ID > ID {
//			fmt.Println("Peer \"", ID, "\" sending ELECTION to: ", p.ID)
//
//			// Send message to p
//			err := send(ID, Utils.ELECTION, p, &reply)
//			if err != nil {
//				log.Fatalln("Error call: ", err)
//			}
//
//			fmt.Println("Peer \"", ID, "\" received from", p.ID, ": ", reply.Msg)
//
//			// If the current peer receive an OK message, it exits the election
//			if reply.Msg == Utils.OK && election {
//				election = false
//				fmt.Println("Peer \"", ID, "\" exit the election")
//			}
//		}
//	}
//}

// Send COORDINATOR message to all peers
//func sendCoordinator() {
//	var reply Utils.Message // Reply message
//
//	// Set coordinator as peer id
//	coordinator = ID
//	fmt.Println("Peer \"", ID, "\" recognized as coordinator himself")
//
//	// Send COORDINATOR to peers
//	for _, p := range peerList {
//		if p.ID != ID {
//			fmt.Println("Peer \"", ID, "\" sending COORDINATOR to: ", p.ID)
//
//			// Send message to p
//			err := send(ID, Utils.COORDINATOR, p, &reply)
//			if err != nil {
//				log.Fatalln("Error call: ", err)
//			}
//		}
//	}
//}

// Check peers status by sending heartbeat message
func heartbeat() {

	// Execute an infinite loop
	for {
		// Repeat every hbtime seconds
		time.Sleep(time.Second * time.Duration(hbtime))

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

	// Repeat send message "tries" times if the send raise an error
	for i := 1; i <= tries; i++ {

		// Random delay in ms
		d := rand.Intn(delay * 1000)
		fmt.Println("Peer \"", ID, "\" generated this delay in ms:", d)
		time.Sleep(time.Duration(d) * time.Millisecond)

		// Connect to the receiver peer
		cli, err := rpc.DialHTTP("tcp", peer.IP+":"+peer.Port)
		if err != nil {
			if i != tries {
				continue
			}
			return err
		}

		// Call the RPC method SendMessage exposed by the receiver peer
		err = cli.Call("Peer.SendMessage", &message, &reply)
		if err != nil {
			if i != tries {
				continue
			}
			return err
		}

		break
	}

	// Return nil if there's no error
	return nil
}
