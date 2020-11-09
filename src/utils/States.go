package utils

import "fmt"

type NodeState struct {
	SendMsg  map[string]Msg
	RecvMsg  []Msg
	NodeName string
	Busy     bool // Node is doing a snapshot
}

func (n NodeState) String() string {
	var res string
	res += "Sent: [ "
	for node, m := range n.SendMsg {
		res += fmt.Sprintf(" %s-> %s,", m.Body, node)
	}
	res += " ], "
	res += "Recv: [ "
	for _, m := range n.RecvMsg {
		res += fmt.Sprintf(" %s,", m.Body)
	}
	res += " ]"
	return res
}

type ChState struct {
	RecvMsg   []Msg
	Recording bool
}

func (cs ChState) String() string {
	var res string
	if cs.Recording {
		res += "Channel is still recording!!!!!!!!!!!!"
	}
	for _, m := range cs.RecvMsg {
		res += fmt.Sprintf(" %s,", m.Body)
	}
	return res
}

type AllState struct {
	Node         NodeState
	Channels     map[string]ChState
	RecvAllMarks bool
}

func (as AllState) String() string {
	res := fmt.Sprintf("\nState: %s", as.Node)
	res += fmt.Sprintf("\nChanels:\n")
	for chKey := range as.Channels {
		res += fmt.Sprintf("[ %s ] ==> %s;", chKey, as.Channels[chKey])
	}
	if !as.RecvAllMarks {
		res += "\n----------------NOT RECEIVED ALL MARKS---------"
	}
	return res
}

type GlobalState struct {
	GS []AllState
}

func (gs GlobalState) String() string {
	res := "\n"
	for _, as := range gs.GS {
		res += "--------------------------------------"
		res += fmt.Sprintf("Node %s: %s\n", as.Node.NodeName, as)
	}
	return res
}
