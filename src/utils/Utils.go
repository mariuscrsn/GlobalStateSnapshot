package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
)

const OutputDirRel= "../output/"
const WorkDirPath  = "/home/cms/Escritorio/uni/redes/practicas/pr1/src/"

type Logger struct {
	// Logs
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

func InitLoggers(name string) *Logger{

	// Initialize log
	fLog, err := os.OpenFile(OutputDirRel+"Log_P"+ name+".log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}

	myLogger := Logger{}
	myLogger.Trace = log.New(fLog,
		"TRACE: \t\t[P"+ name +"] ", log.Ltime| log.Lmicroseconds | log.Lshortfile)

	myLogger.Info = log.New(fLog,
		"INFO: \t\t[P"+ name +"] ", log.Ltime| log.Lmicroseconds | log.Lshortfile)

	myLogger.Warning = log.New(fLog,
		"WARNING: \t[P"+ name +"] ", log.Ltime| log.Lmicroseconds | log.Lshortfile)

	myLogger.Error = log.New(fLog,
		"ERROR: \t\t[P"+ name +"] ", log.Ltime| log.Lmicroseconds | log.Lshortfile)

	return &myLogger
}


func ReadConfig() NetLayout {
	// read file
	data, err := ioutil.ReadFile(WorkDirPath + "network.json")
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
