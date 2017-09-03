package yhdbt

import (
	"fmt"
	"log"
	"sync"
)

const (
	max_players = 4
)

type DeskMnager struct {
	arrPlayers []*PlayerInfo
	DeskNum    int
	muxDesk    sync.Mutex
}

func (this *DeskMnager) InitDesk(id int) {
	this.DeskNum = id
	this.arrPlayers = make([]*PlayerInfo, max_players)
}

func (this *DeskMnager) AddPlayer(p *PlayerInfo) error {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()
	for i, v := range this.arrPlayers {
		if v == nil {
			log.Println(`[DESK] player`, p.NickName, `add desk:`, this.DeskNum, `site:`, i)
			this.arrPlayers[i] = p
			return nil
		}
	}
	return fmt.Errorf(`Desk Full`)
}

func (this *DeskMnager) LeavePlayer(p *PlayerInfo) {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()
	for i, v := range this.arrPlayers {
		if p.Session == v.Session {
			this.arrPlayers[i] = nil
		}
	}
}
