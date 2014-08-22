package mylib

import (
	"github.com/stathat/consistent"
)

type Sender struct {
	Port  int
	Host  string
	Pipe  chan string
	Index int
}

type Boss struct {
	Senders   []Sender
	Rf        int
	Ring      *consistent.Consistent
	Single    int
	Port      string
	DeltaChan chan string
}

type Mmon struct {
	Send int32
	Rcv  int32
	Conn int32
}
