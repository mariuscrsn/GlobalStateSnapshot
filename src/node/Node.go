package node

import (
	"fmt"
	"globalSnapshot/src/utils"
	//"globalSnapshot/src/github.com/DistributedClocks/GoVector/govec"
	"net"
	"strconv"
	"time"
)

const (
	Period       = 2000 * time.Millisecond
)

type Node struct {
	MyNodeInfo      utils.Node
	NetLayout    	utils.NetLayout
	AllState 		utils.AllState
	Listener     	net.Listener
	ChCurrentState	chan utils.AllState
	ChRecvState		chan utils.AllState
	ChRecvMark		chan utils.Msg
	ChSendMark		chan utils.Msg
	ChSendAppMsg	chan utils.OutMsg
	ChRecvAppMsg	chan utils.Msg
	Logger 			*utils.Logger
}

type ErrorNode struct {
	Detail string
}

func (e *ErrorNode) Error() string {
	return e.Detail
}

func NewNode(idxNet int, chRecvAppMsg chan utils.Msg, chSendAppMsg chan utils.OutMsg, chRecvMark chan utils.Msg, chSendMark chan utils.Msg, chCurrentState chan utils.AllState, chRecvState chan utils.AllState, logger *utils.Logger) *Node {

	// Read Network Layout
	var netLayout utils.NetLayout
	netLayout = utils.ReadConfig()
	if len(netLayout.Nodes) < idxNet+1 {
		panic("At least " + strconv.Itoa(idxNet+1) + " processes are needed")
	}

	var myNode = netLayout.Nodes[idxNet]

	// Create channels state
	chsState := make(map[string]utils.ChState)
	for idx, node := range netLayout.Nodes {
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
		MyNodeInfo:     myNode,
		NetLayout:    	netLayout,
		Listener:     	listener,
		AllState: 		utils.AllState{},
		ChCurrentState: chCurrentState,
		ChRecvState: 	chRecvState,
		ChSendAppMsg: 	chSendAppMsg,
		ChRecvAppMsg: 	chRecvAppMsg,
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
	var recvData [1024]byte
	
	for {
		n.Logger.Trace.Println("Waiting for connection accept...")
		if conn, err = n.Listener.Accept(); err != nil {
			n.Logger.Error.Panicf("Server accept connection error: %s", err)
		}
		nBytes, err := conn.Read(recvData[0:])
		fmt.Println(nBytes)
		if err != nil {
			fmt.Printf("%v\n%s\n", conn, conn)
			n.Logger.Error.Panicf("Server accept connection error: %s", err)
		}
		
		if n.AllState.RecvAllMarks { // Waiting for states of the rest of the nodes
			var tempState = utils.AllState{}
			//n.Logger.GoVector.UnpackReceive("Receiving State", recvData[0:nBytes], &tempState, govec.GetDefaultLogOptions())
			// Send state to snapshot
			n.Logger.Info.Printf("Recv State from: %s\n", tempState.Node.NodeName)
			n.ChRecvState <- tempState
		} else
		{ // Waiting for MSG or marks
			var tempMsg utils.Msg = utils.Msg{
				SrcName: "P*",
				Body:    utils.BodyMark,
			}
			//n.Logger.GoVector.UnpackReceive("Receiving Message", recvData[0:nBytes], &tempMsg, govec.GetDefaultLogOptions())
			// Send data to snapshot
			if tempMsg.Body == utils.BodyMark {
				n.Logger.Info.Printf("MARK Recv from: %s\n", tempMsg.SrcName)
				n.ChRecvMark <- tempMsg
			} else {
				n.ChRecvAppMsg <- tempMsg
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
	//opts := govec.GetDefaultLogOptions()
	var outBuf []byte
	outBuf = []byte{'A','B'}
	for {
		select {
		case detMsg := <-n.ChSendAppMsg:
			if !n.AllState.Node.Busy { // it is not performing a global snapshot
				msg := detMsg.Msg
				msg.SrcName = n.MyNodeInfo.Name

				//outBuf := n.Logger.GoVector.PrepareSend("Sending msg", msg, opts)
				if err := n.sendGroup(outBuf, &detMsg); err != nil {
					n.Logger.Error.Panicf("Cannot send app msg: %s", err)
				}
			}
		case <-n.ChSendMark:
			// Block app msg if not blocked yet
			if n.ChSendMark != nil {
				chAux = n.ChSendAppMsg
				n.ChSendMark = nil
			}
			// Send mark
			//outBuf := n.Logger.GoVector.PrepareSend("Sending mark", utils.Msg{
			//	SrcName: n.MyNodeInfo.Name,
			//	Body:    utils.BodyMark,
			//}, opts)
			err := n.sendGroup(outBuf, nil)
			if err != nil {
				n.Logger.Error.Panicf("Cannot send initial mark: %s", err)
			}
		case state := <-n.ChCurrentState:
			n.AllState = state
			if !n.AllState.Node.Busy { // Restart app msg sending
				n.ChSendAppMsg = chAux
			}
			n.Logger.Info.Println("Node state updated")
			if n.AllState.RecvAllMarks {
				n.Logger.Info.Println("Sending my state to all")
				//outBuf := n.Logger.GoVector.PrepareSend("Sending my state to all", n.AllState, opts)
				if err := n.sendGroup(outBuf, nil); err != nil {
					n.Logger.Error.Panicf("Cannot send app msg: %s", err)
				}
			}
		}
	}
}

// Sends req to the group
func (n* Node) sendGroup(data []byte, outMsg *utils.OutMsg) error {
	if outMsg == nil { // sending state
		for _, node := range n.NetLayout.Nodes {
			if node.Name != n.MyNodeInfo.Name {
				go n.sendDirectMsg(data, node, 0)
			}
		}
	} else { // sending msg
		if n.AllState.Node.Busy && outMsg.Msg.Body!= utils.BodyMark {
			return &ErrorNode{"Cannot send msg while global snapshot process is running"}
		}

		for i, idxNode := range outMsg.IdxDest {
			node := n.NetLayout.Nodes[idxNode]
			if node.Name != n.MyNodeInfo.Name {
				go n.sendDirectMsg(data, node, outMsg.Delays[i])
			}
		}
	}
	return nil
}

func (n* Node) sendDirectMsg(msg []byte, node utils.Node, delay int) {
	var conn net.Conn
	var err error

	netAddr := fmt.Sprint(node.IP+":"+strconv.Itoa(node.Port))
	conn, err = net.Dial("tcp", netAddr)
	for i := 0; err != nil && i < n.NetLayout.AttemptsSend; i++ {
		n.Logger.Warning.Printf("Client connection error: %s", err)
		time.Sleep(Period)
		conn, err = net.Dial("tcp", netAddr)
	}
	if err != nil || conn == nil {
		n.Logger.Error.Panicf("Client connection error: %v", err)
	}
	time.Sleep(time.Duration(delay) * time.Millisecond)
	_, err = conn.Write(msg)
	if err != nil{
		n.Logger.Error.Panicf("Sending data error: %v", err)
	}
	err = conn.Close()
	if err != nil {
		n.Logger.Error.Panicf("Clossing connection error: %v", err)
	}
}