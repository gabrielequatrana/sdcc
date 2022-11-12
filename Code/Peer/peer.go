package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"prog/Utils"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/phayes/freeport"
)

type PeerApi int // Used to publish RPC method

var ID int                // Peer id
var peerList []Utils.Peer // List of peers in the network
var numPeer int           // Original number of peers in the network
var conf Utils.Conf       // Configuration of peer and register service
var ip, port string       // IP address and port of the peer
var verbose = false       // Verbose flag

var coordinator int // ID of the coordinator peer
var delay int       // Maximum delay to send a message in ms
var hbTime int      // Repetition interval of heartbeat service
var hbPeer int      // ID of the peer that can run the heartbeat service

var ch chan Utils.Message // Go channel to manage messages
var hbCh chan int         // Go channel to manage heartbeat messages

var election bool // Used only by Bully algorithm. If true, the peer is part of an election
var ring []int    // Used only by Ring algorithm. Contains the peers that are part of the election
var alg bool      // If true then Bully algorithm, else Ring algorithm
var crash bool    // Used in test execution. If true the peer will crash

// TODO Si possono avere shell per ogni peer?

// Peer main
func main() {

	log.Println("Peer service startup, reading config and .env files")

	// Set randomizer seed
	rand.Seed(time.Now().UnixNano())

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Load env file error:", err)
	}

	// Setting verbose flag
	if os.Getenv("VERBOSE") == "1" {
		verbose = true
	}

	// Setting delay
	delay, err = strconv.Atoi(os.Getenv("DELAY"))
	if err != nil {
		log.Fatalln("AtoI delay error:", err)
	}

	// Setting algorithm type
	var a Algorithm
	switch os.Getenv("ALGO") {
	case "bully":
		alg = Utils.BULLY
		a = Bully{}
	case "ring":
		alg = Utils.RING
		a = Ring{}
	}

	// Setting heartbeat time
	hbTime, err = strconv.Atoi(os.Getenv("HEARTBEAT"))
	if err != nil {
		log.Fatalln("AtoI heartbeat time error:", err)
	}

	// Make GO channels
	ch = make(chan Utils.Message)
	hbCh = make(chan int)

	// Reading config file to retrieve IP address and port
	j, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalln("Open config file error:", err)
	}

	// Unmarshalling json file
	err = json.Unmarshal(j, &conf)
	if err != nil {
		log.Fatalln("Unmarshal configuration file error:", err)
	}

	// Registering RPC API
	err = rpc.RegisterName("Peer", new(PeerApi))
	rpc.HandleHTTP()

	// Connect to register service
	regIP := conf.Register.IP
	regPort := conf.Register.Port
	addr := regIP + ":" + regPort
	cli, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatalln("Error dial with register service:", err)
	}

	// Set random port
	p, err := freeport.GetFreePort()
	if err != nil {
		log.Fatalln("GetFreePort error:", err)
	}

	// Open connection
	ip = conf.Peer.IP
	port = strconv.Itoa(p)
	lis, err := net.Listen("tcp", ip+":"+port)
	if err != nil {
		log.Fatalln("Listen error:", err)
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
		log.Fatalln("Error call RegisterPeer:", err)
	}

	// Setting peer ID and retrieve information about other peers
	ID = reply.ID
	peerList = reply.Peers
	numPeer = len(peerList)
	Utils.Print(verbose, "Register service assigned to this peer the id:", ID)
	err = cli.Close()
	if err != nil {
		log.Fatalln("Error close connection with register service:", err)
	}

	// Set crash flag
	peersCrash := strings.Split(os.Getenv("CRASH"), ";")
	for _, pp := range peersCrash {

		// Get peer ID
		pID, e := strconv.Atoi(pp)
		if e != nil {
			log.Fatalln("AtoI crash peer error:", e)
		}

		// Check if the peer will crash
		if pID == ID {
			crash = true
			Utils.Print(verbose, "Peer", ID, "will crash later")
		}
	}

	// Goroutine for serve RPC request coming from other peers
	go func() {
		err = http.Serve(lis, nil)
		if err != nil {
			log.Fatalln("Serve error:", err)
		}
	}()

	// Initially the peer with higher id starts the election to reduce messages number
	if ID == peerList[len(peerList)-1].ID {
		newElection(a)
	}

	// Goroutine for HeartBeat monitoring
	// The peer 0 will start with heartbeat service
	hbPeer = 0
	go heartbeat()

	// Infinite loop executed by peer.
	// Wait for message received in channel and call functions.
	for {
		select {

		// Peer received an ELECTION message
		case msg := <-ch:

			// Check algorithm type
			if alg == Utils.BULLY {
				// If the peer receive an ELECTION message it has to create a new election because it has higher ID
				newElection(a)

			} else if alg == Utils.RING {
				// Check if the peer is already in the election
				if searchElement(msg.ID, ID) {
					// Check if the peer has started the election
					if msg.ID[0] == ID {
						// Send COORDINATOR message
						sort.Ints(msg.ID)
						coordinator = msg.ID[len(msg.ID)-1]
						Utils.Print(verbose, "Peer", ID, "started the election", ring, "and "+
							"found the coordinator:", coordinator)
						a.sendCoordinator()

						// Check crash flag
						if crash {
							os.Exit(0)
						}

					} else {
						// The peer that started the election crashed, then start a new election
						newElection(a)
					}

					// Reset ring
					ring = nil

				} else {
					// Send election to the next peer
					a.sendElection()
				}
			}

		// Peer received an HEARTBEAT message
		case id := <-hbCh:

			// Peer with id is down
			Utils.Print(verbose, "Peer", ID, "know that peer", id, "is down")

			// If the peer down is the coordinator make a new election
			if id == coordinator && !election {
				newElection(a)
			}
		}
	}
}

// SendMessage RPC method provided by peers
func (t *PeerApi) SendMessage(args *Utils.Message, reply *Utils.Message) error {

	// Flag used to check if the peer needs to send a reply
	replyFlag := false

	// Check type of message received
	switch args.Msg {

	// ELECTION message
	case Utils.ELECTION:

		// Check algorithm type
		if alg == Utils.BULLY {
			Utils.Print(verbose, "Peer", ID, "received ELECTION from", args.ID[0])
			replyFlag = true     // Peer needs to send OK message
			reply.Msg = Utils.OK // Send OK message in response
			ch <- *args          // Send message to channel
		} else if alg == Utils.RING {
			Utils.Print(verbose, "Peer", ID, "received ELECTION from", args.ID[len(args.ID)-1])
			if !searchElement(ring, ID) {
				Utils.Print(verbose, "Peer", ID, "joined the election:", append(args.ID, ID))
			}
			ring = args.ID // Add peer id to the election
			ch <- *args    // Send message to channel
		}

	// COORDINATOR message
	case Utils.COORDINATOR:
		Utils.Print(verbose, "Peer", ID, "recognized", args.ID[0], "as coordinator")

		// Reset ring if using ring algorithm
		if alg == Utils.RING {
			ring = nil
		}

		// Set coordinator ID
		coordinator = args.ID[0]

		// Check crash flag non coordinator peer
		if crash {
			os.Exit(0)
		}

	// HEARTBEAT message
	case Utils.HEARTBEAT:
		Utils.Print(verbose, "Peer", ID, "received HEARTBEAT from", args.ID[0])

		// Set reply msg parameters
		reply.ID = []int{ID}
		replyFlag = true // Peer needs to send HEARTBEAT message back
		reply.Msg = Utils.HEARTBEAT
	}

	// Random delay in ms generated only if the peer needs to send a reply
	if replyFlag {
		randomDelay()
	}

	// No error to manage
	return nil
}

// Start a new election in Bully algorithm
func newElection(algorithm Algorithm) {
	algorithm.sendElection()
	if (alg == Utils.BULLY) && election {
		algorithm.sendCoordinator()

		// Check crash flag bully coordinator
		if crash {
			os.Exit(0)
		}
	}
}

// Check peers status by sending heartbeat message
func heartbeat() {

	// Execute an infinite loop
	for {
		// Repeat every hbTime*numPeer seconds
		time.Sleep(time.Second * time.Duration(hbTime))

		// Check if the peer has to run heartbeat service
		if hbPeer == ID {
			Utils.Print(verbose, "Peer", ID, "started heartbeat service")

			// Send heartbeat to all peers
			for i := 0; i <= len(peerList)-1; i++ {
				p := peerList[i]
				beat := new(Utils.Message)
				if p.ID != ID {
					Utils.Print(verbose, "Peer", ID, "sending HEARTBEAT to", p.ID)

					// Send heartbeat to p, if p crashed send ERROR to heartbeat channel
					err := send([]int{ID}, Utils.HEARTBEAT, p, beat)
					if err != nil {
						// If the p is not responding, delete it from the list
						Utils.Print(verbose, "Peer", ID, "not received BEAT response from", p.ID)
						peerList = removeElement(peerList, p)
						i--
						hbCh <- p.ID
					}

					// If the peer responds than it is alive
					if beat.Msg == Utils.HEARTBEAT {
						Utils.Print(verbose, "Peer", ID, "says", beat.ID[0], "is alive")
					}
				}
			}
		}

		// The next peer will run heartbeat service
		hbPeer = (hbPeer + 1) % numPeer
	}
}

// Send a message to a specific peer
func send(id []int, msg int, peer Utils.Peer, reply *Utils.Message) error {

	// Make a new message to send
	message := Utils.Message{
		ID:  id,
		Msg: msg,
	}

	// Wait a random delay
	randomDelay()

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

// Generate random delay in ms
func randomDelay() {
	if delay != 0 {
		d := rand.Intn(delay)
		Utils.Print(verbose, "Peer", ID, "generated this delay in ms:", d)
		time.Sleep(time.Duration(d) * time.Millisecond)
	}
}

// Remove a peer from a slice of peers
func removeElement(slice []Utils.Peer, peer Utils.Peer) []Utils.Peer {
	for i := 0; i <= len(slice)-1; i++ {
		if slice[i] == peer {
			slice = append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// Search an int from a slice of int
func searchElement(slice []int, id int) bool {
	for i := 0; i <= len(slice)-1; i++ {
		if slice[i] == id {
			return true
		}
	}
	return false
}
