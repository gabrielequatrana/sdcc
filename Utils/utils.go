package Utils

import (
	"log"
)

// Algorithm type
const (
	BULLY = true
	RING  = false
)

// Message type
const (
	ELECTION = iota
	OK
	COORDINATOR
	HEARTBEAT
)

// Message struct
type Message struct {
	ID  []int
	Msg int
}

// Peer struct
type Peer struct {
	ID   int
	IP   string
	Port string
}

// RegistrationReply struct
type RegistrationReply struct {
	Peers []Peer
	ID    int
}

// Conf struct
type Conf struct {
	Register struct {
		IP   string `json:"ip"`
		Port string `json:"port"`
	} `json:"register"`
	Peer struct {
		IP   string `json:"ip"`
		Port string `json:"port"`
	} `json:"peer"`
}

// Print check the verbose flag and print in the command line
func Print(verbose bool, a ...any) {
	if verbose {
		log.Println(a...)
	}
}
