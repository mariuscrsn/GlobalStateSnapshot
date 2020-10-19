package utils

import (
	"vclock"
)

const BodyMark = "Mark"

type Delays struct {
	Delays []int
}

func NewDelays() *Delays {
	return &Delays{make([]int, 0)}
}

type Msg struct {
	SrcName string
	Clock   vclock.VClock
	Body    interface{}
}

type OutMsg struct {
	Msg 	Msg
	IdxDest 	int
}

func NewMark(srcName string, clock vclock.VClock) Msg {
	return Msg{SrcName: srcName, Clock: clock, Body: BodyMark}
}

func NewMsg(srcName string, clock vclock.VClock, body interface{}) Msg {
	return Msg{SrcName: srcName, Clock: clock, Body: body}
}

// func (msgArray MsgArray) String() string {
// 	var str string
// 	for _, m := range msgArray.Msgs {
// 		str += fmt.Sprintf("From: %v; body: \"%s\"clk: %v\n", m.SrcName, m.Body, m.Clock)
// 	}
// 	return str
// }
