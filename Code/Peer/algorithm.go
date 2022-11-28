package main

import (
	"log"
	"prog/Utils"
)

// Algorithm interface that define the methods of the two algorithms of distributed election
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
	for i := 0; i <= len(peerList)-1; i++ {

		// If the peer has exited the election, it does not send any more messages
		if !election {
			break
		}

		p := peerList[i]
		if p.ID > ID {

			// Send message to p
			Utils.Print(v, "Peer", ID, "sending ELECTION to", p.ID)
			err := send([]int{ID}, Utils.ELECTION, p, &reply)
			if err != nil {
				// Peer offline
				Utils.Print(v, "Peer", ID, "can't contact", p.ID)
				continue
			}

			// If the current peer receive an OK message, it exits the election
			Utils.Print(v, "Peer", ID, "received OK message from", p.ID)
			if reply.Msg == Utils.OK && election {
				election = false
				Utils.Print(v, "Peer", ID, "exits the election")
			}
		}
	}
}

// SendElection method of Ring Algorithm
func (r Ring) sendElection() {
	var reply Utils.Message // Reply message

	// Append to the election the peer id
	ring = append(ring, ID)

	// Send election message to the next peer in the ring
	for i := 1; i <= len(peerList); i++ {

		// Get the next peer on the ring from the list
		peer := peerList[(ID+i)%len(peerList)]

		// If the next peer on the list is the peer itself, break the loop
		if peer.ID == ID {
			if i == 1 {
				Utils.Print(v, "Peer", ID, "is the only one in the ring so it's the coordinator")
				coordinator = ID
			}
			break
		}

		// Send message to the peer
		Utils.Print(v, "Peer", ID, "sending ELECTION to", peer.ID)
		err := send(ring, Utils.ELECTION, peer, &reply)
		if err != nil {
			// Peer offline, try contacting the next one on the ring
			Utils.Print(v, "Peer", ID, "can't contact", peer.ID, "try to contact next one on the ring")
			continue
		}

		break
	}
}

// SendCoordinator method of Bully Algorithm
func (b Bully) sendCoordinator() {
	var reply Utils.Message // Reply message

	// Set coordinator as peer id
	coordinator = ID
	log.Println("Peer", ID, "recognized as COORDINATOR itself")

	// Send COORDINATOR to peers
	for i := 0; i <= len(peerList)-1; i++ {
		p := peerList[i]
		if p.ID != ID {
			Utils.Print(v, "Peer", ID, "sending COORDINATOR to", p.ID)

			// Send message to p
			err := send([]int{ID}, Utils.COORDINATOR, p, &reply)
			if err != nil {
				// Peer offline
				Utils.Print(v, "Peer", ID, "can't contact", p.ID)
				continue
			}
		}
	}
}

// SendCoordinator method of Ring Algorithm
func (r Ring) sendCoordinator() {
	var reply Utils.Message // Reply message

	log.Println("Peer", ID, "started the election:", ring)

	if coordinator == ID {
		log.Println("Peer", ID, "recognized as COORDINATOR itself")
	} else {
		log.Println("Peer", ID, "recognized as COORDINATOR", coordinator)
	}

	// Send COORDINATOR to peers
	for i := 0; i <= len(peerList)-1; i++ {
		p := peerList[i]
		if p.ID != ID {
			Utils.Print(v, "Peer", ID, "sending COORDINATOR to", p.ID)

			// Send message to p
			err := send([]int{coordinator}, Utils.COORDINATOR, p, &reply)
			if err != nil {
				// Peer offline
				Utils.Print(v, "Peer", ID, "can't contact", p.ID)
				continue
			}
		}
	}
}
