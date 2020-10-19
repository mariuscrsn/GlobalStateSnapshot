package utils

type NodeState struct {
	SendMsg 	[]Msg
	NodeName	string
	Busy		bool // Node is doing a snapshot
}

type ChState struct {
	RecvMsg 	[]Msg
	Recording 	bool
	Sender		string
	Recv		string
}

type AllState struct {
	Node 			NodeState
	Channels 		map[string]ChState
	RecvAllMarks 	bool
}

type GlobalState struct {
	GS 		[]AllState
}