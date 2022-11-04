package main

import (
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
			Utils.Print(verbose, "Peer", ID, "sending ELECTION to", p.ID)

			// Send message to p
			err := send([]int{ID}, Utils.ELECTION, p, &reply)
			if err != nil {
				continue // Peer crashed
			}

			// If the current peer receive an OK message, it exits the election
			Utils.Print(verbose, "Peer", ID, "received OK message from", p.ID)
			if reply.Msg == Utils.OK && election {
				election = false
				Utils.Print(verbose, "Peer", ID, "exits the election")
			}
		}
	}
}

// SendElection method of Ring Algorithm
func (r Ring) sendElection() {
	var reply Utils.Message // Reply message

	// Append to the election the peer id
	ring = append(ring, ID)

	// Send election message to the next peer in the
	for i := 1; i <= len(peerList); i++ {
		peerID := (ID + i) % len(peerList) // ID of the next peer

		// If the next peer on the list is the peer himself, break the loop
		if peerID == ID {
			break
		}

		// Get the peer struct from the list
		peer := peerList[peerID]

		// Send message to the peer
		Utils.Print(verbose, "Peer", ID, "sending ELECTION to", peer.ID)
		err := send(ring, Utils.ELECTION, peer, &reply)
		if err != nil {
			Utils.Print(verbose, "Peer", ID, "can't contact", peer.ID, "\nTry to contact next one on the ring")
			continue // If cant contact the peer, try contacting the next one on the ring
		}

		break
	}
}

// SendCoordinator method of Bully Algorithm
func (b Bully) sendCoordinator() {
	var reply Utils.Message // Reply message

	// Set coordinator as peer id
	coordinator = ID
	Utils.Print(verbose, "Peer", ID, "recognized as COORDINATOR itself")

	// Send COORDINATOR to peers
	for _, p := range peerList {
		if p.ID != ID {
			Utils.Print(verbose, "Peer", ID, "sending COORDINATOR to", p.ID)

			// Send message to p
			err := send([]int{ID}, Utils.COORDINATOR, p, &reply)
			if err != nil {
				continue // Peer crashed
			}
		}
	}
}

// SendCoordinator method of Ring Algorithm
func (r Ring) sendCoordinator() {
	var reply Utils.Message // Reply message

	// Send COORDINATOR to peers
	for _, p := range peerList {
		if p.ID != ID {
			Utils.Print(verbose, "Peer", ID, "sending COORDINATOR to", p.ID)

			// Send message to p
			err := send([]int{coordinator}, Utils.COORDINATOR, p, &reply)
			if err != nil {
				continue // Peer crashed
			}
		}
	}
}
