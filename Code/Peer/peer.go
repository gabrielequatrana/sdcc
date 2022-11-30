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

var ID int                // Peer ID
var peerList []Utils.Peer // List of peers in the network
var numPeer int           // Number of peers in the network
var conf Utils.Conf       // Configuration of peer and register service
var ip, port string       // IP address and port of the peer
var v = false             // Verbose flag
var vv = false            // Full verbose flag (include debug information about delay)

var coordinator int // ID of the coordinator peer
var delay int       // Maximum delay to send a message in ms
var hbTime int      // Duration of the shift of the heartbeat service
var hbPeer int      // ID of the peer that can run the heartbeat service

var ch chan Utils.Message // Go channel to handle messages
var hbCh chan int         // Go channel to handle heartbeat messages
var crCh chan int         // Go channel to handle peer crash during tests

var election bool // Used only by Bully algorithm. If true, the peer is part of an election
var ring []int    // Used only by Ring algorithm. Contains the peers that are part of the election
var alg bool      // If true then Bully algorithm, else Ring algorithm
var crash bool    // Used in test execution. If true the peer will crash

func main() {

	log.Println("Peer service startup, reading config and .env files.")

	// Set randomizer seed
	rand.Seed(time.Now().UnixNano())

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalln("Load env file error:", err)
	}

	// Setting v flag
	if os.Getenv("VERBOSE") == "1" {
		v = true
	}

	// Setting vv flag
	if os.Getenv("VERBOSE") == "2" {
		v = true
		vv = true
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
	crCh = make(chan int)

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

	// Register RPC method
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
	Utils.Print(v, "Register service assigned to this peer the id:", ID)
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
			Utils.Print(v, "Peer", ID, "will crash later.")
		}
	}

	// Goroutine for serve RPC request coming from other peers
	go func() {
		err = http.Serve(lis, nil)
		if err != nil {
			log.Fatalln("Serve error:", err)
		}
	}()

	// Initially the peer with lower id starts the election
	if ID == peerList[0].ID {
		newElection(a)
	}

	// Goroutine for HeartBeat monitoring
	// The peer 0 will start with heartbeat service
	hbPeer = 0
	go heartbeat()

	// Infinite loop executed by peer.
	for {

		// Wait for message received by channels.
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

						a.sendCoordinator()

						// Check crash flag for Ring algorithm
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
			Utils.Print(v, "Peer", ID, "know that peer", id, "is down.")

			// If the coordinator crashed start a new election
			if id == coordinator && !election {
				newElection(a)
			}

		// Peer has to crash in this test
		case <-crCh:
			os.Exit(0)
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
			Utils.Print(v, "Peer", ID, "received ELECTION from", args.ID[0])
			replyFlag = true     // Peer needs to send OK message
			reply.Msg = Utils.OK // Send OK message as reply
			ch <- *args          // Send message to channel
		} else if alg == Utils.RING {
			Utils.Print(v, "Peer", ID, "received ELECTION from", args.ID[len(args.ID)-1])
			if !searchElement(ring, ID) {
				Utils.Print(v, "Peer", ID, "joined the election:", append(args.ID, ID))
			}
			ring = args.ID // Copy the election pool to the peer
			ch <- *args    // Send message to channel
		}

	// COORDINATOR message
	case Utils.COORDINATOR:

		if args.ID[0] == ID {
			log.Println("Peer", ID, "recognized itself as COORDINATOR.")
		} else {
			log.Println("Peer", ID, "recognized", args.ID[0], "as COORDINATOR.")
		}

		// Reset ring if using ring algorithm
		if alg == Utils.RING {
			ring = nil
		}

		// Set coordinator ID
		coordinator = args.ID[0]

		// Check crash flag non coordinator peer
		if crash {
			crCh <- 0
		}

	// HEARTBEAT message
	case Utils.HEARTBEAT:
		Utils.Print(vv, "Peer", ID, "received HEARTBEAT from", args.ID[0])

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
	log.Println("Peer", ID, "is starting a new election.")
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
		// Repeat every hbTime*numPeers seconds
		time.Sleep(time.Second * time.Duration(hbTime))

		// Check if the peer has to run heartbeat service
		if hbPeer == ID {
			log.Println("Peer", ID, "started heartbeat service.")

			// Send heartbeat message to all peers
			for i := 0; i <= len(peerList)-1; i++ {
				p := peerList[i]
				beatReply := new(Utils.Message)
				if p.ID != ID {
					Utils.Print(vv, "Peer", ID, "sending HEARTBEAT to", p.ID)

					// Send heartbeat to p
					err := send([]int{ID}, Utils.HEARTBEAT, p, beatReply)
					if err != nil {
						// If p crashed send ERROR to heartbeat channel
						Utils.Print(vv, "Peer", ID, "not received HEARTBEAT reply from", p.ID)
						hbCh <- p.ID
					}

					// If the peer responds than it is alive
					if beatReply.Msg == Utils.HEARTBEAT {
						Utils.Print(v, "Peer", ID, "says", beatReply.ID[0], "is alive.")
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
		Utils.Print(vv, "Peer", ID, "generated this delay in ms:", d)
		time.Sleep(time.Duration(d) * time.Millisecond)
	}
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
