package node

import (
	"encoding/gob"
	"fmt"
	"net"
	"strconv"
	"time"
	"utils"
	"vclock"
)

const (
	Period       = 2000 * time.Millisecond
)

type Node struct {
	Clk          	vclock.VClock
	MyNodeInfo      utils.Node
	NetLayout    	utils.NetLayout
	AllState 		utils.AllState
	Listener     	net.Listener
	ChState			chan utils.AllState
	ChRecvMark		chan utils.Msg
	ChSendMark		chan utils.Msg
	ChAppMsg		chan utils.OutMsg
	Logger 			*utils.Logger
}

type ErrorNode struct {
	Detail string
}

func (e *ErrorNode) Error() string {
	return e.Detail
}

func NewNode(idxNet int, chAppMsg chan utils.OutMsg, chRecvMark chan utils.Msg, chSendMark chan utils.Msg, chStateSnap chan utils.AllState, logger *utils.Logger) *Node {

	// Read Network Layout
	var netLayout utils.NetLayout
	netLayout = utils.ReadConfig()
	if len(netLayout.Nodes) < idxNet+1 {
		panic("At least " + strconv.Itoa(idxNet+1) + " processes are needed")
	}

	var myNode = netLayout.Nodes[idxNet]

	// Initialize vClock and create channels state
	clk := vclock.New()
	chsState := make(map[string]utils.ChState)
	for idx, node := range netLayout.Nodes {
		clk.Set(node.Name, 0)
		if idx != idxNet {
			chsState[node.Name] = utils.ChState{
				RecvMsg:   make([]utils.Msg, 0),
				Recording: false,
				Sender: myNode.Name,
				Recv:	node.Name,
			}
		}
	}

	// Open Listener port
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(myNode.Port))
	if err != nil {
		panic(fmt.Sprintf("ERROR: unable to open port: %s. Error: %s.", strconv.Itoa(myNode.Port), err))
	}

	var tempNode = Node{
		Clk:          	clk,
		MyNodeInfo:     myNode,
		NetLayout:    	netLayout,
		Listener:     	listener,
		AllState: 		utils.AllState{},
		ChState: 		chStateSnap,
		ChAppMsg: 		chAppMsg,
		ChRecvMark: 	chRecvMark,
		ChSendMark: 	chSendMark,
		Logger: 		logger,
	}

	tempNode.Logger.Trace.Printf("Listening on port: %s", strconv.Itoa(myNode.Port))
	go tempNode.receiver()
	go tempNode.sender()
	return &tempNode
}

func (n* Node) receiver() *utils.Msg {
	var conn net.Conn
	var err error
	var decoder *gob.Decoder

	for {
		n.Logger.Trace.Println("Waiting for connection accept...")
		if conn, err = n.Listener.Accept(); err != nil {
			n.Logger.Error.Panicf("Server accept connection error: %s", err)
		}
		decoder = gob.NewDecoder(conn)
		if n.AllState.RecvAllMarks { // Waiting for states of the rest of the nodes
			var tempState = utils.AllState{}
			if err = decoder.Decode(&tempState); err != nil {
				n.Logger.Error.Panicf("Decoding data error: %s", err)
			}
			// Send state to snapshot
			n.Logger.Info.Printf("Recv State from: %s\n", tempState.Node.NodeName)
			n.ChState <- tempState //TODO: send through other channel
		} else { // Waiting for MSG or marks
			var tempMsg = utils.Msg{}
			if err = decoder.Decode(&tempMsg); err != nil {
				n.Logger.Error.Panicf("Decoding data error: %s", err)
			}
			// Send data to snapshot
			if tempMsg.Body == utils.BodyMark {
				//n.Logger.Info.Printf("MARK Recv from: %s\n", n.MyNodeInfo.Name, tempMsg.SrcName)
				n.ChRecvMark <- tempMsg
			} else {
				// TODO: merge clocks
				n.ChAppMsg <- utils.OutMsg{Msg: tempMsg, IdxDest: 0}
				n.Logger.Info.Printf("Msg Recv: %s\t From: %s\n", tempMsg.Body, tempMsg.SrcName)
				if n.AllState.Node.Busy {
					n.ChRecvMark <- tempMsg // Send msg to snapshot for channel recording
				}
			}
		}
	}
}

func (n *Node) sender(){
	var chAux chan utils.OutMsg
	for {
		select {
		case detMsg := <-n.ChAppMsg:
			if !n.AllState.Node.Busy { // it is not performing a global snapshot
				// Increment local clk before send event
				n.Clk.Tick(n.MyNodeInfo.Name)
				msg := detMsg.Msg
				msg.SrcName = n.MyNodeInfo.Name
				msg.Clock = n.Clk.Copy()
				go n.sendDirectMsg(msg, n.NetLayout.Nodes[detMsg.IdxDest], n.NetLayout.AttemptsSend)
			}
		case mark := <-n.ChSendMark:
			// Block app msg if not blocked yet
			if n.ChAppMsg != nil {
				chAux = n.ChAppMsg
				n.ChAppMsg = nil
			}
			// Send mark
			// TODO: Should I increment clk on mark sending?
			mark = utils.NewMark(n.MyNodeInfo.Name, n.Clk.Copy())
			err := n.sendGroup(&mark, nil)
			if err != nil {
				n.Logger.Error.Panicf("Cannot send initial mark: %s", err)
			}
		case state := <-n.ChState:
			n.AllState = state
			if !n.AllState.Node.Busy { // Restart app msg sending
				n.ChAppMsg = chAux
			}
			n.Logger.Info.Println("Node state updated")
		}
	}
}

// Sends req to the group
func (n* Node) sendGroup(req *utils.Msg, resp *utils.Msg) error {
	if n.AllState.Node.Busy && req.Body!= utils.BodyMark {
		return &ErrorNode{"Cannot send msg while global snapshot process is running"}
	}
	// Increment local clk before send event
	n.Clk.Tick(n.MyNodeInfo.Name)
	msg := utils.NewMsg(n.MyNodeInfo.Name, n.Clk.Copy(), req.Body)

	for _, node := range n.NetLayout.Nodes {
		if node.Name != n.MyNodeInfo.Name {
			go n.sendDirectMsg(msg, node, n.NetLayout.AttemptsSend)
		}
	}
	return nil
}

func (n* Node) sendDirectMsg(msg utils.Msg, node utils.Node, tries int) {
	var conn net.Conn
	var err error
	var encoder *gob.Encoder

	netAddr := fmt.Sprint(node.IP+":"+strconv.Itoa(node.Port))
	conn, err = net.Dial("tcp", netAddr)
	for i := 0; err != nil && i < tries; i++ {
		n.Logger.Warning.Printf("Client connection error: %s", err)
		time.Sleep(Period)
		conn, err = net.Dial("tcp", netAddr)
		if i == tries {
			n.Logger.Error.Panicf("Client connection error: %s", err)
		}
	}

	encoder = gob.NewEncoder(conn)
	n.Logger.Trace.Printf("Msg sent to %s: %s CLK: %s\n", netAddr, msg.Body, msg.Clock)
	err = encoder.Encode(msg)
	if err != nil {
		n.Logger.Error.Panicf("Sending data error: %s", err)
	}
	err = conn.Close()
	if err != nil {
		n.Logger.Error.Panicf("Clossing connection error: %s", err)
	}
}


/*
func isSequentialCLK(localClk vclock.VClock, recvClk vclock.VClock, senderName string) bool {

	var found = true
	var localIndClk, recvIndClk uint64
	localIndClk, ok := localClk.FindTicks(senderName)
	found = found && ok
	recvIndClk, ok = recvClk.FindTicks(senderName)
	found = found && ok
	// Only changes remote sender clk
	if found && (localIndClk+1 == recvIndClk) { // sender clk event increment
		for nodeName, localIndClk := range localClk.GetMap() {
			recvIndClk, ok = recvClk.FindTicks(nodeName)
			found = found && ok && (localIndClk == recvIndClk) // is sequential if the rest of clk_i are equal
		}
	} else {
		found = false
	}
	return found
}


func (n* Node) searchNextMsg() (*utils.Msg, bool) {

	for idx, recvMsg := range n.UnOrderedMsg {
		if isSequentialCLK(n.Clk, recvMsg.Clock, recvMsg.SrcName) {
			// remove delivered element
			n.UnOrderedMsg[idx] = n.UnOrderedMsg[len(n.UnOrderedMsg)-1]
			n.UnOrderedMsg = n.UnOrderedMsg[:len(n.UnOrderedMsg)-1]
			return recvMsg, true
		}
	}
	return nil, false
}

func (n* Node) ReceiveGroup(req *utils.Delays, resp *utils.Msg) error {
	to do complete it
	var tempMsg *utils.Msg
	var found = false
	// Wait until found next msg on queue or receive one
	for !found {
		if len(n.UnOrderedMsg) > 0 {
			if tempMsg, found = n.searchNextMsg(); found {
				break
			}
		}
		// Wait for msg arrive
		tempMsg = n.waitMsg()
		fmt.Print("Después de waitMSG")
		fmt.Println(tempMsg.Body)
		if isSequentialCLK(n.Clk, tempMsg.Clock, tempMsg.SrcName) {
			fmt.Println("sale xk es siguiente")
			break
		}
	}

	// Update local VCLK with arrived msg
	n.Clk.Merge(tempMsg.Clock)

	// Deliver MSG
	resp = tempMsg
	fmt.Println(tempMsg.Body)
	fmt.Println(resp.Body)

	return nil
}
*/
