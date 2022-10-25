package Utils

// Message type
const (
	ELECTION = iota
	OK
	COORDINATOR
	HEARTBEAT
	CLOSE
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
