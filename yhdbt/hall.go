package yhdbt

import (
	"sync"
)

type HallManger struct {
	Desknum   int
	Playernum int
	muxHall   sync.Mutex
}

var GHall = &HallManger{}

func (this *HallManger) Start() {
	this.Desknum = 0
	this.Playernum = 0
}

func (this *HallManger) AddPlayer(p *PlayerInfo) {

}
