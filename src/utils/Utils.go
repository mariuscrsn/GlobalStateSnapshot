package utils

import (
	"encoding/json"
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
)

const OutputDirRel = "output/"
const WorkDirPath = "/home/cms/Escritorio/uni/redes/practicas/pr1/"

//const WorkDirPath = "/home/a721609/Desktop/redes/pr1/"

type Logger struct {
	// Logs
	Trace    *log.Logger
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	GoVector *govec.GoLog
}

func InitLoggers(name string) *Logger {

	// Initialize log
	fLog, err := os.OpenFile(OutputDirRel+"Log_P"+name+".log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}

	myLogger := Logger{}
	myLogger.Trace = log.New(fLog,
		"TRACE: \t\t[P"+name+"] ", log.Ltime|log.Lmicroseconds|log.Lshortfile)

	myLogger.Info = log.New(fLog,
		"INFO: \t\t[P"+name+"] ", log.Ltime|log.Lmicroseconds|log.Lshortfile)

	myLogger.Warning = log.New(fLog,
		"WARNING: \t[P"+name+"] ", log.Ltime|log.Lmicroseconds|log.Lshortfile)

	myLogger.Error = log.New(fLog,
		"ERROR: \t\t[P"+name+"] ", log.Ltime|log.Lmicroseconds|log.Lshortfile)

	//Initialize GoVector logger
	config := govec.GetDefaultConfig()
	config.UseTimestamps = true
	myLogger.GoVector = govec.InitGoVector("P"+name, OutputDirRel+"GoVector/LogFileP"+name, config)

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

func RunRPCSnapshot(conn *rpc.Client, chResp chan GlobalState) {
	go func() {
		var gs GlobalState
		err := conn.Call("App.MakeSnapshot", nil, &gs)
		if err != nil {
			panic(err)
		}
		chResp <- gs
	}()
}

func RunRPCCommand(method string, conn *rpc.Client, content interface{}, resp int, chResp chan int) {
	err := conn.Call(method, &content, nil)
	if err != nil {
		panic(err)
	}
	chResp <- resp
}
