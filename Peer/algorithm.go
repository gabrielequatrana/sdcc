package main

import (
	"fmt"
	"log"
	"prog/Utils"
)

type Algorithm interface {
	sendElection()
	sendCoordinator()
}

type Bully struct{}
type Ring struct{}

func (b Bully) sendElection() {
	var reply Utils.Message // Reply message
	election = true         // The current peer take part in the election

	// Send ELECTION to peers
	for _, p := range peerList {
		if p.ID > ID {
			fmt.Println("Peer \"", ID, "\" sending ELECTION to: ", p.ID)

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

func (r Ring) sendElection() {

}

func (b Bully) sendCoordinator() {
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

func (r Ring) sendCoordinator() {

}
