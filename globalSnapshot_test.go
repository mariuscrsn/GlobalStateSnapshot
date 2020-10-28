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

//func TestEmptySnapshot(t *testing.T) {
//	utils.RunRPCCommand("App.MakeSnapshot", RPCConn["P0"], nil)
//}

func TestMsgAndSnapshot(t *testing.T) {
	msg := utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P0", Body: "MS1"},
		IdxDest: []int{1},
		Delays:  []int{0},
	}
	NMsg := 5
	NSnap := 1
	chRespMsg := make(chan int, NMsg)
	chRespSnap := make(chan int, NSnap)
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P0"], msg, 1, chRespMsg)
	fmt.Println("Test: ordered 1st msg")
	msg.Msg = utils.Msg{
		SrcName: "P2",
		Body:    "MSG2",
	}
	msg.Delays = []int{0}
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P2"], msg, 2, chRespMsg)
	fmt.Println("Test: ordered 2nd msg")
	//time.Sleep(3*time.Second)

	go utils.RunRPCCommand("App.MakeSnapshot", RPCConn["P0"], nil, 1, chRespSnap)
	fmt.Println("Test: ordered GS")
	msg = utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P1", Body: "MS3"},
		IdxDest: []int{0},
		Delays:  []int{0},
	}
	fmt.Println("Test: ordered 3rd msg")
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P1"], msg, 3, chRespMsg)
	msg = utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P1", Body: "MS4"},
		IdxDest: []int{2},
		Delays:  []int{0},
	}
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P1"], msg, 4, chRespMsg)
	fmt.Println("Test: ordered 4th msg")
	msg = utils.OutMsg{
		Msg:     utils.Msg{SrcName: "P0", Body: "MS5 - last"},
		IdxDest: []int{2},
		Delays:  []int{0},
	}
	go utils.RunRPCCommand("App.SendMsg", RPCConn["P1"], msg, 5, chRespMsg)
	fmt.Println("Test: ordered 5th msg")
	for i := 0; i < NMsg; i++ {
		nMsg := <-chRespMsg
		fmt.Printf("Msg nº: %d sent\n", nMsg)
	}
	fmt.Println("All msg sent")
	_ = <-chRespSnap
	fmt.Println("Snapshot completed")

	//time.Sleep(2*time.Second)
	//fmt.Println("Test: ordering last GS")
	//utils.RunRPCCommand("App.MakeSnapshot", RPCConn["P1"], nil)
	//fmt.Println("Test: ordered last GS")
}
