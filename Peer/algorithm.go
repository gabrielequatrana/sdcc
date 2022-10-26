package main

import (
	"fmt"
	"log"
	"prog/Utils"
)

// Algorithm interface that define the two method of the 2 algorithm of distributed election
type Algorithm interface {
	sendElection()
	sendCoordinator()
}

// Bully and Ring are structs that implements the Algorithm interface methods
type Bully struct{}
type Ring struct{}

// SendElection method of Bully Algorithm
func (b Bully) sendElection() {
	var reply Utils.Message // Reply message
	election = true         // The current peer take part in the election

	// Send ELECTION to peers
	for _, p := range peerList {
		if p.ID > ID {
			fmt.Println("Peer \"", ID, "\" sending ELECTION to: ", p.ID)

			// Send message to p
			err := send([]int{ID}, Utils.ELECTION, p, &reply)
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

// SendElection method of Ring Algorithm
func (r Ring) sendElection() {
	var reply Utils.Message
	fmt.Println(reply)

	ring = append(ring, ID)

	// Send election message to the next peer in the
	for i := 1; i <= len(peerList); i++ {
		peerID := (ID + i) % len(peerList)
		if peerID == ID {
			fmt.Println("EXIT CYCLE") // TODO
			break
		}

		peer := peerList[peerID]
		fmt.Println("SEUM:", peerID)

		err := send(ring, Utils.ELECTION, peer, &reply)
		if err != nil {
			continue // If cant contact next peer in the ring, try to contact the other next?
		}

		break
	}
}

// SendCoordinator method of Bully Algorithm
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
			err := send([]int{ID}, Utils.COORDINATOR, p, &reply)
			if err != nil {
				log.Fatalln("Error call: ", err)
			}
		}
	}
}

// SendCoordinator method of Ring Algorithm
func (r Ring) sendCoordinator() {
	var reply Utils.Message // Reply message

	// Set coordinator as peer id
	fmt.Println("Peer \"", ID, "\" recognized as coordinator:", coordinator)

	// Send COORDINATOR to peers
	for _, p := range peerList {
		if p.ID != ID {
			fmt.Println("Peer \"", ID, "\" sending COORDINATOR to: ", p.ID)

			// Send message to p
			err := send([]int{coordinator}, Utils.COORDINATOR, p, &reply)
			if err != nil {
				log.Fatalln("Error call: ", err)
			}
		}
	}
}
