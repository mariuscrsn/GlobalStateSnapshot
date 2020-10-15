package main

import (
	"fmt"
	"os"
	"strconv"
	"globalSnapshot"
	"time"
	"utils"
)

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
	chGS := make(chan utils.GlobalState)
	node := globalSnapshot.InitNode(idx, chGS)
	if idx == 0 {
		node.MakeSnapshot()
	}

	// TODO: delete timeout to finish
	boom := time.After(10000 * time.Millisecond)
	var gs utils.GlobalState
	select {
		case gs = <- chGS:
			fmt.Println(gs)
			return
		case <- boom:
			fmt.Println("BOOM!")
			return
	}
	// c := causalGCom.NewComm(idx)
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
