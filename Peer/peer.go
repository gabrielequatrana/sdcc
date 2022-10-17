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

var msg Utils.Message
var election bool
var coordinator int
var ch chan int

func main() {

	ch = make(chan int)
	fmt.Println("Peer service startup")

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

	// Serve rpc request coming from other peer in a goroutine
	go func() {
		err := http.Serve(lis, nil)
		if err != nil {
			log.Fatalln("Error serve: ", err)
		}
	}()

	// HeartBeat monitoring
	var beat Utils.Message
	go func() {
		for {
			time.Sleep(time.Second * 30) // Repeat every two second
			for _, p := range peerList {
				if p.ID != ID {
					fmt.Println("Peer \"", ID, "\" sending heartbeat to: ", p.ID)
					message := Utils.Message{
						ID:  ID,
						Msg: Utils.HEARTBEAT,
					}

					cli, err := rpc.DialHTTP("tcp", p.IP+":"+p.Port)
					if err != nil {
						log.Fatalln("Error DialHTTP: ", err)
					}

					err = cli.Call("Peer.SendMessage", &message, &beat)
					if err != nil {
						log.Fatalln("Error call: ", err)
					}

					fmt.Println("Peer \"", ID, "\" received from", p.ID, ": ", beat.Msg)
					if beat.Msg == Utils.HEARTBEAT {
						fmt.Println("Peer \"", ID, "\" says", beat.ID, "is alive")
					}
				}
			}
		}
	}()

	// Send initial ELECTION to all
	//sendElection()
	//sendCoordinator()

	// Initially all Peer know the coordinator
	if ID == peerList[len(peerList)-1].ID {
		sendCoordinator()
	}

	select {
	case msg1 := <-ch:
		fmt.Println("SELECTED CASE ELECTION", msg1)
	}

	select {}
}

func (t *api) SendMessage(args *Utils.Message, reply *Utils.Message) error {

	msg := args.Msg

	switch msg {
	case Utils.ELECTION:
		fmt.Println("Peer \"", ID, "\" received: ELECTION from:", args.ID)
		ch <- msg
		reply.Msg = Utils.OK

	case Utils.OK:
		fmt.Println("Peer \"", ID, "\" received: OK")
		reply.Msg = Utils.OK

	case Utils.COORDINATOR:
		fmt.Println("Peer \"", ID, "\" received: COORDINATOR from:", args.ID)
		coordinator = args.ID
		fmt.Println("Peer \"", ID, "\" recognized as coordinator", args.ID)

	case Utils.HEARTBEAT:
		fmt.Println("Peer \"", ID, "\" received: HEARTBEAT from:", args.ID)
		reply.Msg = ID
		reply.Msg = Utils.HEARTBEAT
	}

	return nil
}

// Send ELECTION message to all peers with greater id
func sendElection() {
	election = true
	for _, p := range peerList {
		if p.ID > ID {
			fmt.Println("Peer \"", ID, "\" sending initial election to: ", p.ID)
			message := Utils.Message{
				ID:  ID,
				Msg: Utils.ELECTION,
			}

			cli, err := rpc.DialHTTP("tcp", p.IP+":"+p.Port)
			if err != nil {
				log.Fatalln("Error DialHTTP: ", err)
			}

			err = cli.Call("Peer.SendMessage", &message, &msg)
			if err != nil {
				log.Fatalln("Error call: ", err)
			}

			fmt.Println("Peer \"", ID, "\" received from", p.ID, ": ", msg.Msg)
			if msg.Msg == Utils.OK && election {
				election = false
				fmt.Println("Peer \"", ID, "\" exit the election")
			}
		}
	}
}

func sendCoordinator() {
	coordinator = ID
	fmt.Println("Peer \"", ID, "\" recognized as coordinator himself")
	for _, p := range peerList[:len(peerList)-1] {
		fmt.Println("Peer \"", ID, "\" sending COORDINATOR to: ", p.ID)
		message := Utils.Message{
			ID:  ID,
			Msg: Utils.COORDINATOR,
		}

		cli, err := rpc.DialHTTP("tcp", p.IP+":"+p.Port)
		if err != nil {
			log.Fatalln("Error DialHTTP: ", err)
		}

		err = cli.Call("Peer.SendMessage", &message, &msg)
		if err != nil {
			log.Fatalln("Error call: ", err)
		}
	}
}
