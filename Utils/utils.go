package Utils

const (
	ELECTION = iota
	OK
	COORDINATOR
	CLOSE
)

type Peer struct {
	ID   int
	IP   string
	Port string
}

type RegistrationReply struct {
	Peers []Peer
	ID    int
}

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
