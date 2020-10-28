package snapshot

import (
	"github.com/DistributedClocks/GoVector/govec"
	"globalSnapshot/src/utils"
)

type SnapNode struct {
	idxNode int
	Nodes   []utils.Node

	NodeState      utils.NodeState
	ChannelsStates map[string]utils.ChState

	ChCurrentState chan utils.AllState
	ChRecvState    chan utils.AllState
	ChRecvMark     chan utils.Msg
	ChSendMark     chan utils.Msg
	ChSendMsg      chan utils.OutMsg
	ChAppGS        chan utils.GlobalState
	ChInternalGs   chan utils.GlobalState
	IsLauncher     bool
	Logger         *utils.Logger
}

func NewSnapNode(idxNet int, chRecvMark chan utils.Msg, chSendMark chan utils.Msg, chSendMsg chan utils.OutMsg, chCurrentState chan utils.AllState, chRecvState chan utils.AllState, netLayout *utils.NetLayout, logger *utils.Logger) *SnapNode {
	var myNode = netLayout.Nodes[idxNet]

	// Create channels state
	chsState := make(map[string]utils.ChState)
	for idx, node := range netLayout.Nodes {
		if idx != idxNet {
			chsState[node.Name] = utils.ChState{
				RecvMsg:   make([]utils.Msg, 0),
				Recording: false,
			}
		}
	}

	snapNode := &SnapNode{
		idxNode:        idxNet,
		Nodes:          netLayout.Nodes,
		ChCurrentState: chCurrentState,
		ChRecvState:    chRecvState,
		ChRecvMark:     chRecvMark,
		ChSendMark:     chSendMark,
		ChSendMsg:      chSendMsg,
		ChInternalGs:   make(chan utils.GlobalState),

		NodeState:      utils.NodeState{SendMsg: make(map[string]utils.Msg), RecvMsg: make([]utils.Msg, 0), NodeName: myNode.Name},
		ChannelsStates: chsState,
		Logger:         logger,
	}
	go snapNode.waitForSnapshot()
	return snapNode
}

func (n *SnapNode) MakeSnapshot() utils.GlobalState {
	n.Logger.Info.Println("Initializing snapshot...")
	// Save node state, all prerecording msg (sent btw | prev-state ---- mark | are store on n.NodeState.SendMsg
	n.IsLauncher = true
	n.NodeState.Busy = true // While Busy cannot send new msg

	// Start channels recording
	for chKey := range n.ChannelsStates {
		n.ChannelsStates[chKey] = utils.ChState{
			RecvMsg:   make([]utils.Msg, 0),
			Recording: true,
		}
	}

	// Update state on
	n.ChCurrentState <- utils.AllState{
		Node:         n.NodeState,
		Channels:     n.ChannelsStates,
		RecvAllMarks: false,
	}

	// Send mark
	n.Logger.Info.Println("Sending first Mark...")
	n.ChSendMark <- utils.NewMark(n.Nodes[n.idxNode].Name)

	gs := <-n.ChInternalGs
	return gs
}

func (n *SnapNode) recvMsgMark(nMarks *int8, msg utils.Msg) bool {
	// Recv a mark
	if msg.Body == utils.BodyMark {
		*nMarks = *nMarks + 1
		mark := msg
		// First mark recv, save process state
		if !n.NodeState.Busy {
			n.Logger.Info.Printf("Recv first MARK from %s\n", mark.SrcName)
			n.NodeState.Busy = true // While Busy cannot send new msg

			// Start channels recording
			for chKey := range n.ChannelsStates {
				n.ChannelsStates[chKey] = utils.ChState{
					RecvMsg:   make([]utils.Msg, 0),
					Recording: true,
				}
			}

			// Send broadcast marks
			n.Logger.Info.Printf("Send broadcast Mark\n")
			n.ChSendMark <- mark
		} else {
			// NOT First mark recv, stop recording channel
			n.Logger.Info.Printf("Recv another MARK from %s\n", mark.SrcName)
		}

		tempChState := n.ChannelsStates[mark.SrcName]
		tempChState.Recording = false
		n.ChannelsStates[mark.SrcName] = tempChState

		if *nMarks == int8(len(n.Nodes)) {
			// Send current state to all
			n.Logger.Info.Printf("Recv all MARKs\n")
			n.Logger.GoVector.LogLocalEvent("Recv all MARKs", govec.GetDefaultLogOptions())
			n.Logger.Info.Println("Sending my state to all")
			n.ChCurrentState <- utils.AllState{
				Node:         n.NodeState,
				Channels:     n.ChannelsStates,
				RecvAllMarks: true,
			}
			return true
		}
	} else { // Recv a msg
		if n.NodeState.Busy {
			// Save msg as postrecording
			chState := n.ChannelsStates[msg.SrcName]
			chState.RecvMsg = append(chState.RecvMsg, msg)
			n.ChannelsStates[msg.SrcName] = chState
		} else { // Save msg on node state
			n.NodeState.RecvMsg = append(n.NodeState.RecvMsg, msg)
		}
	}
	return false
}

func (n *SnapNode) recvAllMarks() {
	// Receive all marks
	var nMarks int8 = 1 // my mark
	for {
		var msg utils.Msg
		select {
		case msg = <-n.ChRecvMark: // Recv mark or msg
			if n.recvMsgMark(&nMarks, msg) {
				return
			}
		case detmsg := <-n.ChSendMsg: // node send msg
			for _, idxDst := range detmsg.IdxDest {
				n.NodeState.SendMsg[n.Nodes[idxDst].Name] = detmsg.Msg
			}
		}
	}
}

func (n *SnapNode) waitForSnapshot() {
	for {
		n.recvAllMarks()

		// Gather global status and send to app
		n.Logger.Info.Println("Beginning to gather states...")
		var gs utils.GlobalState
		gs.GS = append(gs.GS, utils.AllState{
			Node:         n.NodeState,
			Channels:     n.ChannelsStates,
			RecvAllMarks: true,
		})
		for i := 0; i < len(n.Nodes)-1; i++ {
			indState := <-n.ChRecvState
			n.Logger.Info.Printf("Recv State from: %s\n", indState.Node.NodeName)
			gs.GS = append(gs.GS, indState)
		}
		n.Logger.Info.Println("All states gathered")

		// Restore process state
		n.NodeState.Busy = false
		n.NodeState.SendMsg = make(map[string]utils.Msg)
		n.NodeState.RecvMsg = make([]utils.Msg, 0)

		// Inform node to continue receiving msg
		n.ChCurrentState <- utils.AllState{
			Node:         n.NodeState,
			Channels:     n.ChannelsStates,
			RecvAllMarks: false,
		}

		// Send gs to launcher
		if n.IsLauncher {
			n.ChInternalGs <- gs
			n.IsLauncher = false
		}
	}
}
