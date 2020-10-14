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

type Comm struct {
	Clk          vclock.VClock
	MyNode       utils.Node
	NetLayout    utils.NetLayout
	Listener     net.Listener
	UnOrderedMsg []*utils.Msg

	// Logs
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func (c *Comm) initLoggers(
	traceHandle io.Writer, // final output trace
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer, addr string) {

	c.Trace = log.New(traceHandle,
		"TRACE: \t\t["+addr+"] ", log.Ltime|log.Lshortfile)

	c.Info = log.New(infoHandle,
		"INFO: \t\t["+addr+"] ", log.Ltime|log.Lshortfile)

	c.Warning = log.New(warningHandle,
		"WARNING: \t["+addr+"] ", log.Ltime|log.Lshortfile)

	c.Error = log.New(errorHandle,
		"ERROR: \t\t["+addr+"] ", log.Ltime|log.Lshortfile)
}

func NewComm(idxNet int) Comm {
	// Read Network Layout
	var netLayout utils.NetLayout
	netLayout = utils.ReadConfig(WorkDirPath + "network.json")
	if len(netLayout.Nodes) < idxNet+1 {
		panic("At least " + strconv.Itoa(idxNet+1) + " processes are needed")
	}

	// Initialize vClock
	clk := vclock.New()
	for _, node := range netLayout.Nodes {
		clk.Set(node.Name, 0)
	}

	var myNode = netLayout.Nodes[idxNet]

	// Open Listener port
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(myNode.Port))
	if err != nil {
		panic(fmt.Sprintf("ERROR: unable to open port: %s. Error: %s.", strconv.Itoa(netLayout.Nodes[idxNet].Port), err))
	}

	// Initialize log
	fLog, err := os.OpenFile(OutputDirRel+"Log_"+myNode.Name, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}

	// gob.Register(utils.Msg{})
	var cTemp = Comm{
		Clk:          clk,
		MyNode:       myNode,
		NetLayout:    netLayout,
		Listener:     listener,
		UnOrderedMsg: make([]*utils.Msg, 0),
	}

	cTemp.initLoggers(fLog, fLog, fLog, fLog, myNode.Name)
	return cTemp
}

func MakeSnapshot() {
	fmt.Println("Initializing snapshot...")
}

// Sends req to the group
func (c *Comm) SendGroup(req *utils.Msg, resp *utils.Msg) error {
	// Increment local clk before send event
	c.Clk.Tick(c.MyNode.Name)
	msg := utils.NewMsg(c.MyNode.Name, c.Clk.Copy(), req.Body)

	for _, node := range c.NetLayout.Nodes {
		if node.Name != c.MyNode.Name {
			go c.sendDirectMsg(msg, &node, c.NetLayout.AttemptsSend)
		}
	}
	return nil
}

func (c *Comm) sendDirectMsg(msg utils.Msg, node *utils.Node, tries int) {
	var conn net.Conn
	var err error
	var encoder *gob.Encoder

	conn, err = net.Dial("tcp", node.IP+":"+strconv.Itoa(node.Port))
	for i := 0; err != nil && i < tries; i++ {
		c.Warning.Printf("Client connection error: %s", err)
		time.Sleep(Period)
		conn, err = net.Dial("tcp", node.IP+":"+strconv.Itoa(node.Port))
		if i == tries {
			c.Error.Panicf("Client connection error: %s", err)
		}
	}

	encoder = gob.NewEncoder(conn)
	c.Info.Printf("[%s] Sent to %s: %s CLK: %s\n", msg.SrcName, node.IP, msg.Body, msg.Clock)
	err = encoder.Encode(msg)
	if err != nil {
		c.Error.Panicf("Sending data error: %s", err)
	}
	err = conn.Close()
	if err != nil {
		c.Error.Panicf("Clossing connection error: %s", err)
	}
}

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

func (c *Comm) searchNextMsg() (*utils.Msg, bool) {

	for idx, recvMsg := range c.UnOrderedMsg {
		if isSequentialCLK(c.Clk, recvMsg.Clock, recvMsg.SrcName) {
			// remove delivered element
			c.UnOrderedMsg[idx] = c.UnOrderedMsg[len(c.UnOrderedMsg)-1]
			c.UnOrderedMsg = c.UnOrderedMsg[:len(c.UnOrderedMsg)-1]
			return recvMsg, true
		}
	}
	return nil, false
}

func (c *Comm) ReceiveGroup(req *utils.Delays, resp *utils.Msg) error {

	var tempMsg *utils.Msg
	var found = false
	// Wait until found next msg on queue or receive one
	for !found {
		if len(c.UnOrderedMsg) > 0 {
			if tempMsg, found = c.searchNextMsg(); found {
				break
			}
		}
		// Wait for msg arrive
		tempMsg = c.waitMsg()
		fmt.Print("Después de waitMSG")
		fmt.Println(tempMsg.Body)
		if isSequentialCLK(c.Clk, tempMsg.Clock, tempMsg.SrcName) {
			fmt.Println("sale xk es siguiente")
			break
		}
	}

	// Update local VCLK with arrived msg
	c.Clk.Merge(tempMsg.Clock)

	// Deliver MSG
	resp = tempMsg
	fmt.Println(tempMsg.Body)
	fmt.Println(resp.Body)

	return nil
}

//func (e *ErrorCausal) Error() string {
//	return fmt.Sprintf(e.What, e.Args)
//}

func (c *Comm) waitMsg() *utils.Msg {
	var conn net.Conn
	var err error
	var decoder *gob.Decoder
	var tempMsg = utils.Msg{}

	// c.Listener, err = net.Listen("tcp", strconv.Itoa(c.MyNode.Port))
	// if err != nil {
	// 	Error.Panicf("Server listen error: %s", err)
	// }

	for {
		c.Trace.Println("Waiting for connection accept...")
		if conn, err = c.Listener.Accept(); err != nil {
			c.Error.Panicf("Server accept connection error: %s", err)
		}
		decoder = gob.NewDecoder(conn)
		if err = decoder.Decode(&tempMsg); err != nil {
			c.Error.Panicf("Decoding data error: %s", err)
		}

		// Send data to manage
		c.Info.Println("[%s] Msg Recv: \t From: %s : ", c.MyNode.Name, tempMsg.Body, tempMsg.SrcName)
		return &tempMsg
	}
}
