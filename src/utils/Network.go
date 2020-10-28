package utils

type Node struct {
	Name    string `json:"Name"`
	User    string `json:"User"`
	IP      string `json:"IP"`
	Port    int    `json:"Port"`
	RPCPort int    `json:"RPCPort"`
	Delays  []int  `json:"Delays"`
}

type NetLayout struct {
	Nodes        []Node `json:"Nodes"`
	AttemptsSend int
}
