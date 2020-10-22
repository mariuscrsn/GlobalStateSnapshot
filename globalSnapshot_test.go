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
	fmt.Println(netLayout.Nodes)

	// Start logger govec
	logger := govec.InitGoVector(netLayout.Nodes[0].Name, utils.OutputDirRel+"GoVector/LogFileClientP0", govec.GetDefaultConfig())
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

func TestEmptySnapshot(t *testing.T) {
	utils.RunRPCCommand("App.MakeSnapshot", RPCConn["P0"], nil)
}

func TestCutMsg(t *testing.T) {

}
