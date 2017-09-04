package yhdbt

import (
	"net"
)

type PlayerInfo struct {
	Session    string
	NickName   string
	Uid        string
	Score      int
	Win        int
	Lose       int
	Run        int
	LastOnline int64
	Conn       net.Conn
	ArrCards   []int
}
