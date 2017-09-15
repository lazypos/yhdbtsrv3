package yhdbt

import (
	"log"
)

const (
	err_code_kicked = 0x7000001
)

type KickedManager struct {
	chKick chan *PlayerInfo
}

var GKicked = &KickedManager{}

func (this *KickedManager) Start() {
	this.chKick = make(chan *PlayerInfo, 100)
	go this.Routine_workTick()
}

func (this *KickedManager) AddTick(p *PlayerInfo) {
	this.chKick <- p
}

func (this *KickedManager) Routine_workTick() {
	for {
		select {
		case player := <-this.chKick:
			//  发送 被踢信息
			log.Println(player)
		}
	}
}
