package yhdbt

import (
	"fmt"
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
	ticker := time.NewTicker(time.Second * 10)
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
	//删除没人的桌子
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	for k, d := range this.MapDesks {
		if d.Empty() {
			delete(this.MapDesks, k)
		}
	}
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

//创建桌子，从大到小创建
func (this *HallManger) CreateDesk(p *PlayerInfo) (*DeskMnager, int) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()

	for i := max_desk; i > 0; i-- {
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

//加入桌子
func (this *HallManger) AddDesk(num int, p *PlayerInfo) (*DeskMnager, int) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()

	if num > max_desk {
		return nil, -1
	}
	desk, ok := this.MapDesks[num]
	if ok {
		site := desk.AddPlayer(p)
		if site != -1 {
			return desk, site
		}
	}
	return nil, -1
}

//快速加入桌子
func (this *HallManger) FastAddDesk(p *PlayerInfo) (*DeskMnager, int) {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()

	//先找有人的桌子
	for _, d := range this.MapDesks {
		site := d.AddPlayer(p)
		if site != -1 {
			return d, site
		}
	}

	//创建新桌子
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

func (this *HallManger) AddScore(n int, uid string) bool {
	this.muxHall.Lock()
	defer this.muxHall.Unlock()
	pinfo, ok := this.MapPlayers[uid]
	if ok && pinfo != nil {
		pinfo.Score += n
		if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_score`, uid)), pinfo.Score); err != nil {
			log.Println(`[充值] save player score error:`, uid, pinfo.Score)
		}
		return true
	}
	Score := GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_score`, uid)))
	Score += n
	log.Println(`uid 当前积分`, Score-n, "充值后：", Score)
	if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_score`, uid)), Score); err != nil {
		log.Println(`[充值] save player score error:`, uid, Score)
	}
	return true
}
