package globalSnapshot

import (
	"fmt"
	"globalSnapshot/utils"
	"net/rpc"
	"os"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	// WorkDirPath = "/home/a721609/redes/pr1/"
	WorkDirPath = "/home/cms/Escritorio/uni/redes/practicas/pr1/src/"
)

var RPCConn map[string]*rpc.Client
var sshConn map[string]*ssh.Client

// Connect via ssh and initialize RPC nodes
func TestMain(m *testing.M) {
	fmt.Println("Starting tests for Causal Group Comm... ")
	setupNodes()
	fmt.Println("Execute the rest of the tests...")
	code := m.Run()
	fmt.Println("Causal Group Comm tests finished. Closing...")
	terminate()
	os.Exit(code)
}

// Source: http://networkbit.ch/golang-ssh-client/
// https://medium.com/tarkalabs/ssh-recipes-in-go-part-one-5f5a44417282
func setupNodes() {

	var netLayout utils.NetLayout
	netLayout = utils.ReadConfig(WorkDirPath + "network.json")
	if len(netLayout.Nodes) < 2 {
		panic("At least 2 processes are needed")
	}
	fmt.Println(netLayout.Nodes)

	sshConn = make(map[string]*ssh.Client, 0)
	RPCConn = make(map[string]*rpc.Client, 0)
	for idx, node := range netLayout.Nodes {
		fmt.Println("Starting: " + node.Name)
		sshConn[node.Name] = utils.ConnectSSH(node.User, node.IP)

		// Initialize RPC

		var cmd = "cd " + WorkDirPath + ";/usr/local/go/bin/go run " + WorkDirPath + "groupCom.go " + strconv.Itoa(idx) + " " + strconv.Itoa(node.RPCPort)
		go utils.RunCommandSSH(cmd, sshConn[node.Name])

		time.Sleep(3000 * time.Millisecond) // Wait for RPC initialization
		// Connect via RPC to the server
		clientRPC, err := rpc.DialHTTP("tcp", node.IP+":"+strconv.Itoa(node.RPCPort))
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
		utils.RunCommandSSH("pkill groupCom", sshConn[name])
		sshConn[name].Close()
	}
}

func TestOrdered(t *testing.T) {
	utils.SendGroup(RPCConn["P1"], "MSG-1")
	checkRestProc(t, "MSG-1", []string{"P2", "P3"})

	// utils.SendGroup(RPCConn["P2"], "MSG-2")
	// checkRestProc(t, "MSG-2", []string{"P1", "P3"})

	// utils.SendGroup(RPCConn["P3"], "MSG-3")
	// checkRestProc(t, "MSG-3", []string{"P1", "P2"})
}

func checkRestProc(t *testing.T, expect string, names []string) {
	for _, proc := range names {
		var recv utils.Msg
		go utils.ReceiveGroup(RPCConn[proc], utils.NewDelays(), recv)
		// if recv.Body != expect {
		// 	t.Errorf("ERROR: [%q]receiveGroup --> recv [%q] !=!=! expect [%q]", proc, recv, expect)
		// }
	}
}
