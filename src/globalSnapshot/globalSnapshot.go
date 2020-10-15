package globalSnapshot

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
	"utils"
	"vclock"
)

const (
	// WorkDirPath = "/home/a721609/redes/pr1/"
	WorkDirPath  = "/home/cms/Escritorio/uni/redes/practicas/pr1/src/"
	Period       = 2000 * time.Millisecond
	OutputDirRel = "../output/"
)

type Node struct {
	Clk          	vclock.VClock
	MyNodeInfo      utils.Node
	NetLayout    	utils.NetLayout
	Listener     	net.Listener
	NodeState		utils.NodeState
	ChannelsStates 	map[string]utils.ChState
	ChGlobState		chan utils.GlobalState
	ChMark		chan utils.Msg

	// Logs
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func (n* Node) initLoggers(
	traceHandle io.Writer, // final output trace
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer, addr string) {

	n.Trace = log.New(traceHandle,
		"TRACE: \t\t["+addr+"] ", log.Ltime|log.Lshortfile)

	n.Info = log.New(infoHandle,
		"INFO: \t\t["+addr+"] ", log.Ltime|log.Lshortfile)

	n.Warning = log.New(warningHandle,
		"WARNING: \t["+addr+"] ", log.Ltime|log.Lshortfile)

	n.Error = log.New(errorHandle,
		"ERROR: \t\t["+addr+"] ", log.Ltime|log.Lshortfile)
}

func InitNode(idxNet int, chGS chan utils.GlobalState) *Node {

	// Read Network Layout
	var netLayout utils.NetLayout
	netLayout = utils.ReadConfig(WorkDirPath + "network.json")
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

	// Initialize log
	fLog, err := os.OpenFile(OutputDirRel+"Log_"+myNode.Name, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}


	// gob.Register(utils.Msg{})
	var tempNode = Node{
		Clk:          	clk,
		MyNodeInfo:     myNode,
		NetLayout:    	netLayout,
		Listener:     	listener,
		NodeState: 		utils.NodeState{SendMsg: make([]utils.Msg, 0), NodeName: myNode.Name},
		ChannelsStates:	chsState,
		ChGlobState: 	chGS,
		ChMark: 		make(chan utils.Msg),
	}

	tempNode.initLoggers(fLog, fLog, fLog, fLog, myNode.Name)
	tempNode.Trace.Printf("Listening on port: %s", strconv.Itoa(myNode.Port))
	go tempNode.waitForSnapshot()
	go tempNode.waitMsg()
	return &tempNode
}

//func NewComm(idxNet int) Comm {
	//TODO: aqui va lo de InitNode
//}

func (n* Node) MakeSnapshot() {
	n.Info.Println("Initializing snapshot...")
	// Save node state, all prerecording msg (sent btw | prev-state ---- mark | are store on n.NodeState.SendMsg
	// Todo change it with channel
	n.NodeState.Busy = true	// While Busy cannot send new msg

	// Send mark
	mark := utils.NewMark(n.MyNodeInfo.Name, n.Clk)
	err := n.sendGroup(&mark, nil)
	if err != nil {
		n.Error.Panicf("Cannot send initial mark: %s", err)
	}
	n.Info.Println("First Mark Sent")

	// Start channels recording
	for _, c := range n.ChannelsStates {
		c.Recording = true
		c.RecvMsg = make([]utils.Msg, 0)
	}
}

func (n*  Node) waitForSnapshot(){
	// Receive all marks
	var nMarks int8 = 0
	for {
		var mark utils.Msg
		mark = <-n.ChMark
		nMarks++

		// First mark recv, save process state
		if !n.NodeState.Busy{
			n.Info.Printf("[%s] Recv fisrt MARK from %s\n", n.MyNodeInfo.Name, mark.SrcName)
			n.NodeState.Busy = true // While Busy cannot send new msg
			// Start channels recording
			for _, c := range n.ChannelsStates {
				if c.Sender != mark.SrcName { // Mark channel not record
					c.Recording = true
				}
				c.RecvMsg = make([]utils.Msg, 0)
			}
			// Send broadcast marks
			n.Trace.Printf("[%s] Send broadcast Mark\n", n.MyNodeInfo.Name)
			err := n.sendGroup(&mark, nil)
			if err != nil {
				n.Error.Panicf("Cannot send mark: %s", err)
			}
		} else {
		// NOT First mark recv, stop recording channel
			n.Trace.Printf("[%s] Recv another MARK from %s\n", n.MyNodeInfo.Name, mark.SrcName)
			tempChState := n.ChannelsStates[mark.SrcName]
			tempChState.Recording = false
			n.ChannelsStates[mark.SrcName] = tempChState
		}

		if nMarks == int8(len(n.NetLayout.Nodes)-1) {
			n.Info.Printf("[%s] Recv all MARKs\n", n.MyNodeInfo.Name)
			break
		}
	}
	// Gather global status and send to app
	//TODO: complet it
	// Restore process state
	//TODO: complet it
}

// Sends req to the group
func (n* Node) sendGroup(req *utils.Msg, resp *utils.Msg) error {
	if n.NodeState.Busy && req.Body!= utils.BodyMark {
		return &ErrorGSS{"Cannot send msg while global snapshot process is running"}
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
		n.Warning.Printf("Client connection error: %s", err)
		time.Sleep(Period)
		conn, err = net.Dial("tcp", netAddr)
		if i == tries {
			n.Error.Panicf("Client connection error: %s", err)
		}
	}

	encoder = gob.NewEncoder(conn)
	n.Trace.Printf("[%s] Msg sent to %s: %s CLK: %s\n", msg.SrcName, netAddr, msg.Body, msg.Clock)
	err = encoder.Encode(msg)
	if err != nil {
		n.Error.Panicf("Sending data error: %s", err)
	}
	err = conn.Close() //TODO: esto está bien así o debería ser un defer?
	if err != nil {
		n.Error.Panicf("Clossing connection error: %s", err)
	}
}

func (n* Node) waitMsg() *utils.Msg {
	var conn net.Conn
	var err error
	var decoder *gob.Decoder
	var tempMsg = utils.Msg{}

	for {
		n.Trace.Println("Waiting for connection accept...")
		if conn, err = n.Listener.Accept(); err != nil {
			n.Error.Panicf("Server accept connection error: %s", err)
		}
		decoder = gob.NewDecoder(conn)
		if err = decoder.Decode(&tempMsg); err != nil {
			n.Error.Panicf("Decoding data error: %s", err)
		}

		// Send data to manage
		if tempMsg.Body == utils.BodyMark {
			n.ChMark <- tempMsg
			n.Info.Printf("[%s] MARK Recv from: %s\n", n.MyNodeInfo.Name, tempMsg.SrcName)
		} else {
			n.Info.Printf("[%s] Msg Recv: %s\t From: %s\n", n.MyNodeInfo.Name, tempMsg.Body, tempMsg.SrcName)
		}
	}
}

type ErrorGSS struct {
	Detail string
}
func (e *ErrorGSS) Error() string {
	return e.Detail
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
	TODO: complete it
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