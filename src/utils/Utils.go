package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/rpc"
)

func ReadConfig(filePath string) NetLayout {
	// read file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	fmt.Println("Reading network configuration file...")

	var netCfg NetLayout
	// parse content of json file to Config struct
	err = json.Unmarshal(data, &netCfg)
	if err != nil {
		panic(err)
	}

	return netCfg
}

func SendGroup(conn *rpc.Client, content interface{}) {
	req := Msg{Body: content}
	resp := Msg{}
	err := conn.Call("Comm.SendGroup", &req, &resp)
	if err != nil {
		panic(err)
	}
}

func ReceiveGroup(conn *rpc.Client, delays *Delays, resp Msg) {
	err := conn.Call("Comm.ReceiveGroup", &delays, &resp)
	if err != nil {
		panic(err)
	}
}
