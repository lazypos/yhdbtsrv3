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
	breakTicker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			this.broadHallInfo() // 广播大厅基本信息
		case msg := <-this.chBroad:
			this.broadMessage(msg) // 广播实时信息
		case <-breakTicker.C:
			this.checkBreak() // 断线检测
		}
	}
}

func (this *HallManger) broadMessage(msg string) {

}

func (this *HallManger) broadHallInfo() {

}

//断线检测
func (this *HallManger) checkBreak() {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()

	nowTime := time.Now().Unix()
	for _, p := range this.MapPlayers {
		if nowTime-p.LastOnline > 180 {
			if p.DeskNum > 0 {
				desk := this.MapDesks[p.DeskNum]
				desk.LeavePlayer(p)
			} else {
				delete(this.MapPlayers, p.Uid)
				log.Println(`[HALL] player leave:`, p.Conn.RemoteAddr().String())
			}
		}
	}
}

// deskNum: <0创建桌子，0加入任意桌子，>0加入桌子 返回桌号 座位号
func (this *HallManger) AddDesk(deskNum int, p *PlayerInfo) (*DeskMnager, int) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()

	//带桌号
	if deskNum > 0 {
		if deskNum > 100 {
			return nil, -1
		}
		desk, ok := this.MapDesks[deskNum]
		if ok {
			site := desk.AddPlayer(p)
			if site != -1 {
				return nil, site
			}
		}
		return nil, -1
	}
	//任意桌子
	if deskNum == 0 {
		//任意桌子
		for _, desk := range this.MapDesks {
			if s := desk.AddPlayer(p); s != -1 {
				return desk, s
			}
		}
	}
	//创建桌子
	for i := 1; i < max_desk; i++ {
		_, ok := this.MapDesks[i]
		if !ok {
			desk := &DeskMnager{}
			desk.InitDesk(i)
			s := desk.AddPlayer(p)
			this.MapDesks[i] = desk
			return desk, s
		}
	}
	return nil, -1
}

func (this *HallManger) CreateDesk() (*DeskMnager, int) {
	return nil, -1
}

func (this *HallManger) BroadDeskInfo(deskNum int) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	desk, _ := this.MapDesks[deskNum]
	if desk != nil {
		desk.ToBroadInfo()
	}
}

func (this *HallManger) LeaveDesk(p *PlayerInfo) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	deks, _ := this.MapDesks[p.DeskNum]
	if deks != nil {
		if deks.LeavePlayer(p) {
			// 桌子空了就删掉
			delete(this.MapDesks, p.DeskNum)
		}
	}
	p.DeskNum = -1
}

func (this *HallManger) GetDesk(Dnum int) *DeskMnager {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	return this.MapDesks[Dnum]
}

func (this *HallManger) QueryPlayerCounts() int {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	return len(this.MapPlayers)
}
