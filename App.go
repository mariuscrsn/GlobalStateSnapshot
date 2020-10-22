package main

import (
	"encoding/gob"
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
	"globalSnapshot/src/node"
	"globalSnapshot/src/snapshot"
	"globalSnapshot/src/utils"
	"net"
	"net/rpc"
	"os"
	"strconv"
)

type App struct {
	snap         *snapshot.SnapNode
	node         *node.Node
	log          *utils.Logger
	chSendAppMsg chan utils.OutMsg
	chRecvAppMsg chan utils.Msg
}

func NewApp(idxNet int) *App {
	var app App
	app.chSendAppMsg = make(chan utils.OutMsg)  // node <--    msg   --- app
	app.chRecvAppMsg = make(chan utils.Msg)     // node ---    msg   --> app
	chRecvMark := make(chan utils.Msg)          // node --- mark|msg --> snap
	chCurrentState := make(chan utils.AllState) // node <-- AllState --- snap
	chRecvState := make(chan utils.AllState)    // node --- AllState --> snap
	chSendMark := make(chan utils.Msg)          // node <-- SendMark --> snap

	// Register struct
	gob.Register(utils.Msg{})
	app.log = utils.InitLoggers(strconv.Itoa(idxNet))
	app.node = node.NewNode(idxNet, app.chRecvAppMsg, app.chSendAppMsg, chRecvMark, chSendMark, chCurrentState, chRecvState, app.log)
	app.snap = snapshot.NewSnapNode(idxNet, chRecvMark, chSendMark, chCurrentState, chRecvState, &app.node.NetLayout, app.log)
	return &app
}

func (a *App) Receiver(rq *interface{}, resp *interface{}) error {
	//tick := time.Tick(100 * time.Millisecond)
	for {
		select {
		//case <- tick:
		//	fmt.Println(a.node.MyNodeInfo.Name + " ")
		case msg := <-a.chRecvAppMsg:
			fmt.Printf("Msg [%v] recv from: %s", msg.Body, msg.SrcName)
		}
	}
}

func (a *App) MakeSnapshot(rq *interface{}, resp *interface{}) error {
	gs := a.snap.MakeSnapshot()
	a.log.Info.Printf("Received global state: %v\n", gs)
	return nil
}

func (a *App) SendMsg(rq *interface{}, resp *interface{}) error {
	var locRq utils.OutMsg
	//TODO: convert it
	fmt.Println("recibida petición")
	a.chSendAppMsg <- locRq
	for idx := range locRq.IdxDest {
		fmt.Printf("Msg [%v] sent to: %s", locRq.Msg.Body, a.node.NetLayout.Nodes[idx].Name)
	}
	return nil
}

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		panic("Incorrect number of arguments. Usage: go run groupCom.go <0-based index node> <RPC port>")
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		panic(fmt.Sprintf("Bad argument[0]: %s. Error: %s. Usage: go run grpCausal.go <0-based index node> <RPC port>", args[0], err))
	}
	fmt.Println("Starting process ", idx)
	myApp := NewApp(idx)
	go myApp.Receiver(nil, nil)
	// Register app as RPC
	server := rpc.NewServer()
	err = server.Register(myApp)
	//err = rpc.Register(myApp)
	if err != nil {
		panic(err)
	}
	rpc.HandleHTTP()
	_, err = strconv.Atoi(args[1])
	if err != nil {
		panic(fmt.Sprintf("Bad argument[1]: %s. Error: %s. Usage: go run grpCausal.go <0-based index node> <RPC port>", args[0], err))
	}
	l, err := net.Listen("tcp", ":"+args[1])
	if err != nil {
		panic(err)
	}
	options := govec.GetDefaultLogOptions()
	vrpc.ServeRPCConn(server, l, myApp.log.GoVector, options)
	//err = http.Serve(l, nil)
	//if err != nil {
	//	panic(err)
	//}
}
