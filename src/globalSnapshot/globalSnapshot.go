package globalSnapshot

import (
	"utils"
)

type SnapNode struct {
	idxNode      	int
	Nodes    		[]utils.Node

	NodeState		utils.NodeState
	ChannelsStates 	map[string]utils.ChState

	ChStateNode		chan utils.AllState
	ChRecvMark		chan utils.Msg
	ChSendMark		chan utils.Msg
	ChAppGS			chan utils.GlobalState
	ChInternalGs	chan utils.GlobalState
	Logger 			*utils.Logger
}



func NewSnapNode(idxNet int, chRecvMark chan utils.Msg, chSendMark chan utils.Msg, chStateNode chan utils.AllState, chGSApp chan utils.GlobalState, netLayout *utils.NetLayout, logger *utils.Logger) *SnapNode {
	var myNode = netLayout.Nodes[idxNet]

	snapNode := &SnapNode{
		idxNode:    	idxNet,
		Nodes:    		netLayout.Nodes,
		ChStateNode: 	chStateNode,
		ChAppGS: 		chGSApp,
		ChRecvMark: 	chRecvMark,
		ChSendMark: 	chSendMark,
		ChInternalGs: 	make(chan utils.GlobalState),

		NodeState: 		utils.NodeState{SendMsg: make([]utils.Msg, 0), NodeName: myNode.Name},
		ChannelsStates: map[string]utils.ChState{},
		Logger:			logger,
	}
	go snapNode.waitForSnapshot()
	return snapNode
}
func (n* SnapNode) MakeSnapshot() utils.GlobalState {
	n.Logger.Info.Println("Initializing snapshot...")
	// Save node state, all prerecording msg (sent btw | prev-state ---- mark | are store on n.NodeState.SendMsg
	n.NodeState.Busy = true	// While Busy cannot send new msg

	// Start channels recording
	for _, c := range n.ChannelsStates {
		c.Recording = true
		c.RecvMsg = make([]utils.Msg, 0)
	}

	// Update state on
	n.ChStateNode <- utils.AllState{
		Node:         n.NodeState,
		Channels:     n.ChannelsStates,
		RecvAllMarks: false,
	}

	// Send mark
	n.Logger.Info.Println("Sending first Mark...")
	n.ChSendMark <- utils.NewMark(n.Nodes[n.idxNode].Name, nil)

	gs := <- n.ChInternalGs
	return gs
}

func (n*  SnapNode) waitForSnapshot(){
	// Receive all marks
	var nMarks int8 = 1 // my mark
	for {
		var mark utils.Msg
		mark = <-n.ChRecvMark
		nMarks++

		// First mark recv, save process state
		if !n.NodeState.Busy{
			n.Logger.Info.Printf("Recv first MARK from %s\n", mark.SrcName)
			n.NodeState.Busy = true // While Busy cannot send new msg

			// Start channels recording
			for _, c := range n.ChannelsStates {
				if c.Sender != mark.SrcName { // Mark channel not record
					c.Recording = true
				}
				c.RecvMsg = make([]utils.Msg, 0)
			}

			// Send broadcast marks
			n.Logger.Trace.Printf("Send broadcast Mark\n")
			n.ChSendMark <- mark
		} else {
			// NOT First mark recv, stop recording channel
			n.Logger.Trace.Printf("Recv another MARK from %s\n", mark.SrcName)
			tempChState := n.ChannelsStates[mark.SrcName]
			tempChState.Recording = false
			n.ChannelsStates[mark.SrcName] = tempChState
		}

		if nMarks == int8(len(n.Nodes)) {
			n.Logger.Info.Printf("Recv all MARKs\n")
			break
		}
	}
	// Gather global status and send to app
	n.Logger.Info.Println("Beginning to gather states...")
	var gs utils.GlobalState
	for i:= 0; i < len(n.Nodes)-1; i++{
		indState := <- n.ChStateNode
		gs.GS = append(gs.GS, indState)
	}
	n.Logger.Info.Println("All states gathered")

	// Restore process state
	n.NodeState.SendMsg = make([]utils.Msg, 0)
	n.NodeState.Busy = false

	n.ChInternalGs <- gs
}