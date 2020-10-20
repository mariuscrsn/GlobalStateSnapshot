package main

import (
	"fmt"
	"snapshot"
	"node"
	"os"
	"strconv"
	"time"
	"utils"
)

type App struct {
	snap 	*snapshot.SnapNode
	node 	*node.Node
	log 	*utils.Logger
	chAppMsg chan utils.OutMsg
}

func NewApp(idxNet int) *App {
	var app App
	app.chAppMsg = make(chan utils.OutMsg)		// node <--   msg  --> snap
	chRecvMark := make(chan utils.Msg)			// node --- mark|msg --> snap
	chCurrentState := make(chan utils.AllState) // node <-- AllState --- snap
	chRecvState := make(chan utils.AllState) 	// node --- AllState --> snap
	chSendMark := make(chan utils.Msg)			// node <-- SendMark --> snap

	app.log = utils.InitLoggers(strconv.Itoa(idxNet))
	app.node = node.NewNode(idxNet, app.chAppMsg, chRecvMark, chSendMark, chCurrentState, chRecvState, app.log)
	app.snap = snapshot.NewSnapNode(idxNet, chRecvMark, chSendMark, chCurrentState, chRecvState, &app.node.NetLayout, app.log)
	return &app
}

func (a *App) MakeSnapshot(){
	gs:= a.snap.MakeSnapshot()
	a.log.Info.Printf("Received global state: %v\n", gs)
}

func main() {
	args := os.Args[1:]
	if len(args) != 1 { // Todo change it
		panic("Incorrect number of arguments. Usage: go run groupCom.go <0-based index node> <RPC port>")
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		panic(fmt.Sprintf("Bad argument[0]: %s. Error: %s. Usage: go run grpCausal.go <0-based index node> <RPC port>", args[0], err))
	}
	fmt.Println("Starting process ", idx)
	app := NewApp(idx)
	if idx == 0 {
		app.MakeSnapshot()

	}

	time.Sleep(20 * time.Second)
	fmt.Println("FINISH!")
	return
	//TODO: chang it for working with RPC

	// // Register c as RPC and serve
	// err = rpc.Register(&c)
	// if err != nil {
	// 	panic(err)
	// }

	// rpc.HandleHTTP()
	// _, err = strconv.Atoi(args[1])
	// if err != nil {
	// 	panic(fmt.Sprintf("Bad argument[1]: %s. Error: %s. Usage: go run grpCausal.go <0-based index node> <RPC port>", args[0], err))
	// }
	// l, err := net.Listen("tcp", ":"+args[1])
	// if err != nil {
	// 	panic(err)
	// }
	// err = http.Serve(l, nil)
	// if err != nil {
	// 	panic(err)
	// }
}