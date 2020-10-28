package node

import (
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"globalSnapshot/src/utils"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Period = 2000 * time.Millisecond
)

type Node struct {
	MyNodeInfo     utils.Node
	NetLayout      utils.NetLayout
	AllState       utils.AllState
	Listener       net.Listener
	ChCurrentState chan utils.AllState
	ChRecvState    chan utils.AllState
	ChRecvMark     chan utils.Msg
	ChSendMark     chan utils.Msg
	ChSendMsg      chan utils.OutMsg
	ChSendAppMsg   chan utils.OutMsg
	ChRecvAppMsg   chan utils.Msg
	Logger         *utils.Logger
	Mutex          sync.Mutex
}

type ErrorNode struct {
	Detail string
}

func (e *ErrorNode) Error() string {
	return e.Detail
}

func NewNode(idxNet int, chRecvAppMsg chan utils.Msg, chSendAppMsg chan utils.OutMsg, chRecvMark chan utils.Msg, chSendMark chan utils.Msg, chSendMsg chan utils.OutMsg, chCurrentState chan utils.AllState, chRecvState chan utils.AllState, logger *utils.Logger) *Node {

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
		NetLayout:      netLayout,
		Listener:       listener,
		AllState:       utils.AllState{},
		ChCurrentState: chCurrentState,
		ChRecvState:    chRecvState,
		ChSendAppMsg:   chSendAppMsg,
		ChRecvAppMsg:   chRecvAppMsg,
		ChRecvMark:     chRecvMark,
		ChSendMark:     chSendMark,
		ChSendMsg:      chSendMsg,
		Logger:         logger,
		Mutex:          sync.Mutex{},
	}

	tempNode.Logger.Trace.Printf("Listening on port: %s", strconv.Itoa(myNode.Port))
	go tempNode.receiver()
	go tempNode.sender()
	return &tempNode
}

func (n *Node) receiver() *utils.Msg {
	var conn net.Conn
	var err error
	var recvData []byte
	recvData = make([]byte, 1024)

	for {
		n.Logger.Trace.Println("Waiting for connection accept...")
		if conn, err = n.Listener.Accept(); err != nil {
			n.Logger.Error.Panicf("Server accept connection error: %s", err)
		}
		nBytes, err := conn.Read(recvData[0:])
		if err != nil {
			n.Logger.Error.Panicf("Server accept connection error: %s", err)
		}
		//TODO: en estas 2 ramas se bloquea en alguna si los canales son síncronos, no buferrizados
		//n.Logger.Info.Printf("Recv data: %s\n", recvData)
		if !strings.Contains(string(recvData[0:nBytes]), "Channels") {
			// Waiting for MSG or marks
			var tempMsg utils.Msg
			n.Logger.GoVector.UnpackReceive("Receiving Message", recvData[0:nBytes], &tempMsg, govec.GetDefaultLogOptions())
			// Send data to snapshot
			if tempMsg.Body == utils.BodyMark {
				n.ChRecvMark <- tempMsg
				n.Logger.Info.Printf("MARK recv from: %s\n", tempMsg.SrcName)
			} else {
				n.Logger.GoVector.LogLocalEvent(fmt.Sprintf("MSG content: %s, from [%s]", tempMsg.Body, tempMsg.SrcName), govec.GetDefaultLogOptions())
				n.ChRecvAppMsg <- tempMsg
				//if locState.Node.Busy {
				n.ChRecvMark <- tempMsg // Send msg to snapshot for channel recording
				//}
				n.Logger.Info.Printf("MSG [%s] recv from: %s\n", tempMsg.Body, tempMsg.SrcName)
			}
		} else {
			//if locState.RecvAllMarks { // Waiting for states of the rest of the nodes
			var tempState = utils.AllState{}
			n.Logger.GoVector.UnpackReceive("Receiving State", recvData[0:nBytes], &tempState, govec.GetDefaultLogOptions())
			n.Logger.Info.Println("State recv from: ", tempState.Node.NodeName)
			// Send state to snapshot
			n.ChRecvState <- tempState
		}
	}
}

func (n *Node) sender() {
	opts := govec.GetDefaultLogOptions()
	var outBuf []byte
	outBuf = []byte{'A', 'B'}
	for {
		select {
		case detMsg := <-n.ChSendAppMsg:
			n.Mutex.Lock()
			locState := n.AllState
			n.Mutex.Unlock()
			if !locState.Node.Busy { // it is not performing a global snapshot
				msg := detMsg.Msg
				msg.SrcName = n.MyNodeInfo.Name
				outBuf = n.Logger.GoVector.PrepareSend(fmt.Sprintf("Sending msg: %s", msg.Body), msg, opts)
				if err := n.sendGroup(outBuf, &detMsg); err != nil {
					n.Logger.Error.Panicf("Cannot send app msg: %s", err)
				}
				n.ChSendAppMsg <- utils.OutMsg{
					Msg:     utils.Msg{},
					IdxDest: nil,
					Delays:  nil,
				}
				n.ChSendMsg <- detMsg

			} else {
				n.Logger.Warning.Println("Cannot send app msg while node is performing global snapshot")
				n.ChSendAppMsg <- detMsg
			}
		case <-n.ChSendMark:
			// Send mark
			mark := utils.Msg{
				SrcName: n.MyNodeInfo.Name,
				Body:    utils.BodyMark}
			outBuf := n.Logger.GoVector.PrepareSend("Sending mark", mark, opts)
			outMsg := utils.OutMsg{
				Msg:     mark,
				IdxDest: nil,
				Delays:  nil,
			}
			err := n.sendGroup(outBuf, &outMsg)
			if err != nil {
				n.Logger.Error.Panicf("Cannot send initial mark: %s", err)
			}
		case state := <-n.ChCurrentState:
			n.Mutex.Lock()
			n.AllState = state
			n.Mutex.Unlock()
			n.Logger.Info.Println("Node state updated")
			if state.RecvAllMarks {
				outBuf := n.Logger.GoVector.PrepareSend("Sending my state to all", state, opts)
				if err := n.sendGroup(outBuf, nil); err != nil {
					n.Logger.Error.Panicf("Cannot send app msg: %s", err)
				}
			}
		}
	}
}

// Sends req to the group
func (n *Node) sendGroup(data []byte, outMsg *utils.OutMsg) error {
	if outMsg == nil { // sending state
		for _, node := range n.NetLayout.Nodes {
			if node.Name != n.MyNodeInfo.Name {
				n.Logger.Info.Printf("Sending state to: %s\n", node.Name)
				go n.sendDirectMsg(data, node, 2)
			}
		}
	} else { // sending mark
		if outMsg.Msg.Body == utils.BodyMark {
			for i, node := range n.NetLayout.Nodes {
				if node.Name != n.MyNodeInfo.Name {
					//fmt.Printf("Sending mark delay: %d\n", n.MyNodeInfo.Delays[i])
					go n.sendDirectMsg(data, node, n.MyNodeInfo.Delays[i])
				}
			}
		} else { // sending msg
			//n.Mutex.Lock()
			//state := n.AllState
			//n.Mutex.Unlock()
			//if state.Node.Busy  {
			//	return &ErrorNode{"Cannot send msg while global snapshot process is running"}
			//}
			for i, idxNode := range outMsg.IdxDest {
				node := n.NetLayout.Nodes[idxNode]
				if node.Name != n.MyNodeInfo.Name {
					go n.sendDirectMsg(data, node, outMsg.Delays[i])
				}
			}
		}
	}
	return nil
}

func (n *Node) sendDirectMsg(msg []byte, node utils.Node, delay int) {
	var conn net.Conn
	var err error

	time.Sleep(time.Duration(delay) * time.Second)
	netAddr := fmt.Sprint(node.IP + ":" + strconv.Itoa(node.Port))
	conn, err = net.Dial("tcp", netAddr)
	for i := 0; err != nil && i < n.NetLayout.AttemptsSend; i++ {
		n.Logger.Warning.Printf("Client connection error: %s", err)
		time.Sleep(Period)
		conn, err = net.Dial("tcp", netAddr)
	}
	if err != nil || conn == nil {
		n.Logger.Error.Panicf("Client connection error: %v", err)
	}
	_, err = conn.Write(msg)
	if err != nil {
		n.Logger.Error.Panicf("Sending data error: %v", err)
	}
	err = conn.Close()
	if err != nil {
		n.Logger.Error.Panicf("Clossing connection error: %v", err)
	}
}
