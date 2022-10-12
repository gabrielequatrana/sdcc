package main

import (
	"encoding/json"
	"fmt"
	"github.com/phayes/freeport"
	"log"
	"net"
	"net/rpc"
	"os"
	"prog/Utils"
	"strconv"
)

var ID int
var peerList []Utils.Peer
var conf Utils.Conf
var ip, port string

func main() {

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

	buf := make([]byte, 1)
	buf[0] = Utils.COORDINATOR

	if ID == peerList[len(peerList)-1].ID {
		for _, pe := range peerList {
			if pe.ID != ID {
				fmt.Println("Peer \"", ID, "\" sending initial election to: ", pe.ID)
				con, err := net.Dial("tcp", pe.IP+":"+pe.Port)
				if err != nil {
					log.Fatalln("Dial error: ", err)
				}
				_, err = con.Write(buf)
				if err != nil {
					log.Fatalln("Write error: ", err)
				}
			}
		}
	}

	for {
		buf2 := make([]byte, 1)
		cop, _ := lis.Accept()
		_, err = cop.Read(buf2)
		if err != nil {
			log.Fatalln("Read error: ", err)
		}

		switch buf2[0] {
		case Utils.COORDINATOR:
			fmt.Println("Peer \"", ID, "\" received: COORDINATOR")
		}
	}
}
