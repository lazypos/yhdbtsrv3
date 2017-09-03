package yhdbt

import (
	"log"
	"net"
)

const (
	err_code_kicked = 0x7000001
)

type KickedManager struct {
	chKick chan net.Conn
}

var GKicked = &KickedManager{}

func (this *KickedManager) Start() {
	this.chKick = make(chan net.Conn, 100)
}

func (this *KickedManager) AddTick(conn net.Conn) {
	this.chKick <- conn
}

func (this *KickedManager) workTick() {
	for {
		select {
		case conn := <-this.chKick:
			//  发送 被踢信息
		}
	}
}
