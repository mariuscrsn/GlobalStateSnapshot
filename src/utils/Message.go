package utils

const BodyMark = "Mark"

type Msg struct {
	SrcName string
	Body    interface{}
}

type OutMsg struct {
	Msg 		Msg
	IdxDest 	[]int
	Delays		[]int // seconds
}

func NewMark(srcName string) Msg {
	return Msg{SrcName: srcName, Body: BodyMark}
}

func NewMsg(srcName string, body interface{}) Msg {
	return Msg{SrcName: srcName, Body: body}
}
