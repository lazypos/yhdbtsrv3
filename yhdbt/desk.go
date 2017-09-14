package yhdbt

import (
	"bytes"
	"fmt"
	"log"
	"sync"
)

const (
	max_players = 4
)

//桌子
type DeskMnager struct {
	arrPlayers  []*PlayerInfo
	DeskNum     int
	muxDesk     sync.Mutex
	chDeskBoard chan bool
}

func (this *DeskMnager) InitDesk(id int) {
	this.chDeskBoard = make(chan bool, 10)
	this.DeskNum = id
	this.arrPlayers = make([]*PlayerInfo, max_players)
	go this.Routine_Board()
}

//玩家加入,返回座位号 -1桌子满
func (this *DeskMnager) AddPlayer(p *PlayerInfo) int {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()
	for i, v := range this.arrPlayers {
		if v == nil {
			log.Println(`[DESK] player`, p.NickName, `add desk:`, this.DeskNum, `site:`, i)
			this.arrPlayers[i] = p
			return i
		}
	}
	return -1
}

//玩家离开,返回是否空桌子
func (this *DeskMnager) LeavePlayer(p *PlayerInfo) bool {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()

	empty := true
	for i, v := range this.arrPlayers {
		if v != nil {
			if p.Session == v.Session {
				this.arrPlayers[i] = nil
			} else {
				empty = false
			}
		}
	}
	return empty
}

func (this *DeskMnager) ToBroadInfo() {
	this.chDeskBoard <- true
}

func (this *DeskMnager) Routine_Board() {
	for {
		select {
		case <-this.chDeskBoard:
			this.broadDeskInfo()
		}
	}
}

//广播桌子信息
func (this *DeskMnager) broadDeskInfo() {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()

	buf := bytes.NewBufferString("")
	//`{"site":"%d","name":"%s","ready":"%d","socre":"%d","win":"%d","lose":"%d","run":"%d"},`
	for i, p := range this.arrPlayers {
		buf.WriteString(fmt.Sprintf(fmt_change_sub, i, p.NickName, p.Ready, p.Win, p.Lose, p.Run))
	}
	buf.Truncate(buf.Len() - 1)
	for _, p := range this.arrPlayers {
		p.SendMessage(fmt.Sprintf(fmt_change, buf.String()))
	}
}
