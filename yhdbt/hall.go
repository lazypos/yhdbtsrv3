package yhdbt

import (
	"log"
	"sync"
	"time"
)

const (
	max_desk = 100
)

//游戏大厅
type HallManger struct {
	muxHall    sync.Mutex
	chBroad    chan string
	MapPlayers map[string]*PlayerInfo //uid
	MapDesks   map[int]*DeskMnager    //桌子
}

var GHall = &HallManger{}

func (this *HallManger) Start() {
	this.chBroad = make(chan string, 1000)
	this.MapPlayers = make(map[string]*PlayerInfo)
	this.MapDesks = make(map[int]*DeskMnager)
	go this.Routine_Broadcast()
}

//玩家加入大厅
func (this *HallManger) AddPlayer(p *PlayerInfo) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	this.MapPlayers[p.Uid] = p
	log.Println(`[HALL] player add:`, p.Conn.RemoteAddr().String())
}

//玩家离开大厅
func (this *HallManger) LeaveHall(p *PlayerInfo) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	delete(this.MapPlayers, p.Uid)
	log.Println(`[HALL] player leave:`, p.Conn.RemoteAddr().String())
}

// 定时广播消息
func (this *HallManger) Routine_Broadcast() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			this.broadHallInfo() // 广播大厅基本信息
		case msg := <-this.chBroad:
			this.broadMessage(msg) // 广播实时信息
		}
	}
}

func (this *HallManger) broadMessage(msg string) {

}

func (this *HallManger) broadHallInfo() {

}

//定时清理无效的桌子
func (this *HallManger) Routine_Clean() {

}

// <0创建桌子，0加入任意桌子，>0加入桌子
func (this *HallManger) AddDesk(deskNum int, p *PlayerInfo) int {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()

	//带桌号
	if deskNum > 0 {
		if deskNum > 100 {
			return err_desk_noexist
		}
		desk, ok := this.MapDesks[deskNum]
		if ok {
			if desk.AddPlayer(p) != nil {
				return 0
			}
			return err_desk_full
		}
		return err_desk_noexist
	}
	//任意桌子
	if deskNum == 0 {
		//任意桌子
		for k, desk := range this.MapDesks {
			if desk.AddPlayer(p) == nil {
				return k
			}
		}
	}
	//创建桌子
	for i := 1; i < max_desk; i++ {
		_, ok := this.MapDesks[i]
		if !ok {
			desk := &DeskMnager{}
			desk.InitDesk(i)
			desk.AddPlayer(p)
			this.MapDesks[deskNum] = desk
			return i
		}
	}
	return err_desk_full
}
