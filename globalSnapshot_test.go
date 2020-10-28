package main_test

import (
	"encoding/gob"
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
	"globalSnapshot/src/utils"
	"golang.org/x/crypto/ssh"
	"net/rpc"
	"os"
	"strconv"
	"testing"
	"time"
)

var RPCConn map[string]*rpc.Client
var sshConn map[string]*ssh.Client

// Connect via ssh and initialize RPC nodes
func TestMain(m *testing.M) {
	fmt.Println("Starting tests for Global Snapshot... ")
	setupNodes()
	fmt.Println("Execute the rest of the tests...")
	code := m.Run()
	fmt.Println("Global Snapshot tests finished. Closing...")
	terminate()
	os.Exit(code)
}

// Source: http://networkbit.ch/golang-ssh-client/
func setupNodes() {
	gob.Register(utils.Msg{})
	var netLayout utils.NetLayout
	netLayout = utils.ReadConfig()
	if len(netLayout.Nodes) < 2 {
		panic("At least 2 processes are needed")
	}
	fmt.Printf("Net layout: %v\n", netLayout.Nodes)
	// Start logger govec
	config := govec.GetDefaultConfig()
	config.UseTimestamps = true
	logger := govec.InitGoVector("T", utils.OutputDirRel+"GoVector/LogFileTest", config)
	sshConn = make(map[string]*ssh.Client, 0)
	RPCConn = make(map[string]*rpc.Client, 0)
	for idx, node := range netLayout.Nodes {
		//fmt.Println("Starting: " + node.Name)
		sshConn[node.Name] = utils.ConnectSSH(node.User, node.IP)

		// Initialize RPC
		var cmd = "cd " + utils.WorkDirPath + ";/usr/local/go/bin/go run " + utils.WorkDirPath + "App.go " + strconv.Itoa(idx) + " " + strconv.Itoa(node.RPCPort)
		go utils.RunCommandSSH(cmd, sshConn[node.Name])

		time.Sleep(3000 * time.Millisecond) // Wait for RPC initialization

		// Connect via RPC to the server
		clientRPC, err := vrpc.RPCDial("tcp", node.IP+":"+strconv.Itoa(node.RPCPort), logger, govec.GetDefaultLogOptions())
		if err != nil {
			panic(err)
		}
		RPCConn[node.Name] = clientRPC
	}
}

func terminate() {
	// var killed_once bool = false
	for name, conn := range RPCConn {
		_ = conn.Close()
		utils.RunCommandSSH("pkill App", sshConn[name])
		_ = sshConn[name].Close()
	}
}

/*
func TestEmptySnapshot(t *testing.T) {
	utils.RunRPCCommand("App.MakeSnapshot", RPCConn["P0"], nil)
}

*/

/*
func TestSimple(t *testing.T) {
	chRespMsg := make(chan int, 1)
	chRespSnap := make(chan utils.GlobalState, 1)
	msg5 := utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P0", Body: "MS5 - last"},
		IdxDest: []int{2},
		Delays:  []int{0},
	}
	utils.RunRPCCommand("App.SendMsg", RPCConn["P0"], msg5, 5, chRespMsg)
	fmt.Println("Test: ordered 5th msg")

	fmt.Println("Test: ordered last GS")
	utils.RunRPCSnapshot( RPCConn["P1"], chRespSnap)
	gs := <-chRespSnap
	fmt.Printf("Snapshot completed: %s\n", gs)
}
*/

func TestMsgAndSnapshot(t *testing.T) {
	NMsg := 4
	chRespMsg := make(chan int, NMsg)
	chRespSnap := make(chan utils.GlobalState, 1)
	msg1 := utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P0", Body: "MS1"},
		IdxDest: []int{1},
		Delays:  []int{0},
	}
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P0"], msg1, 1, chRespMsg)
	fmt.Println("Test: ordered 1st msg")

	msg2 := utils.OutMsg{
		Msg: utils.Msg{
			SrcName: "P2",
			Body:    "MSG2"},
		IdxDest: []int{1},
		Delays:  []int{0},
	}
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P2"], msg2, 2, chRespMsg)
	fmt.Println("Test: ordered 2nd msg")

	time.Sleep(2 * time.Second)
	go utils.RunRPCSnapshot(RPCConn["P0"], chRespSnap)
	fmt.Println("Test: ordered GS")

	time.Sleep(4 * time.Second)
	msg3 := utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P1", Body: "MS3"},
		IdxDest: []int{0},
		Delays:  []int{0},
	}
	utils.RunRPCCommand("App.SendMsg", RPCConn["P1"], msg3, 3, chRespMsg)
	fmt.Println("Test: ordered 3rd msg")

	msg4 := utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P1", Body: "MS4"},
		IdxDest: []int{2},
		Delays:  []int{0},
	}
	utils.RunRPCCommand("App.SendMsg", RPCConn["P1"], msg4, 4, chRespMsg)
	fmt.Println("Test: ordered 4th msg")

	for i := 0; i < NMsg; i++ {
		nMsg := <-chRespMsg
		fmt.Printf("Msg nº: %d sent\n", nMsg)
	}
	fmt.Println("All msg sent")
	gs := <-chRespSnap
	fmt.Printf("Snapshot completed: %s\n", gs)

	msg5 := utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P0", Body: "MS5 - last"},
		IdxDest: []int{2},
		Delays:  []int{0},
	}
	utils.RunRPCCommand("App.SendMsg", RPCConn["P0"], msg5, 5, chRespMsg)
	fmt.Println("Test: ordered 5th msg")

	time.Sleep(10 * time.Second)
	fmt.Println("Test: ordered last GS")
	utils.RunRPCSnapshot(RPCConn["P1"], chRespSnap)
	gs = <-chRespSnap
	fmt.Printf("Snapshot completed: %s\n", gs)
}
