package yhdbt

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	max_players = 4
)

type DeskMsg struct {
	Text string
	Site int
}

//桌子
type DeskMnager struct {
	arrPlayers  []*PlayerInfo //玩家列表
	DeskNum     int           //桌好
	bPlaying    bool          //是否在游戏
	MapAddTimes map[int]int64 //玩家加入桌子的时间
	baseScore   int           //基础分

	muxDesk     sync.Mutex
	chDeskBoard chan bool
	chDeskMsg   chan *DeskMsg
}

func (this *DeskMnager) InitDesk(id int) {
	this.chDeskBoard = make(chan bool, 10)
	this.chDeskMsg = make(chan *DeskMsg, 100)
	this.MapAddTimes = make(map[int]int64)
	this.DeskNum = id
	this.bPlaying = false
	this.baseScore = 10
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
			p.DeskNum = this.DeskNum
			this.arrPlayers[i] = p
			this.MapAddTimes[i] = time.Now().Unix()
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
	if !this.bPlaying { // 如果游戏还没开始
		for i, v := range this.arrPlayers {
			if v != nil {
				if p.Session == v.Session {
					p.DeskNum = -1
					this.arrPlayers[i] = nil
					this.MapAddTimes[i] = 0
				} else {
					empty = false
				}
			}
		}
	} else { //游戏已经开始
		this.playerRun(p)
		return false
	}
	return empty
}

// 玩家逃跑
func (this *DeskMnager) playerRun(p *PlayerInfo) {
	p.Run += 1
	if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_run`, p.Uid)), p.Run); err != nil {
		log.Println(`[DESK] save player run error:`, p.Uid, p.Run)
	}
	this.arrPlayers[p.SiteNum] = nil
	p.DeskNum = -1
	for _, v := range this.arrPlayers {
		if v != nil {
			v.SendMessage(fmt.Sprintf(fmt_run, p.SiteNum, p.NickName, this.baseScore))
		}
	}
	p.Score -= this.baseScore
	if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_score`, p.Uid)), p.Score); err != nil {
		log.Println(`[DESK] save player score error:`, p.Uid, p.Score)
	}
	this.gameOver(true, []int{})
}

// 游戏结束
func (this *DeskMnager) gameOver(run bool, arrRst []int) {
	//更新状态
	this.bPlaying = false
	for i, p := range this.arrPlayers {
		this.MapAddTimes[i] = 0
		if p != nil {
			p.Ready = 0
		}
	}
	if run {
		return
	}
	//广播信息
	buf := bytes.NewBufferString("")
	for i := 0; i < 4; i++ {
		buf.WriteString(fmt.Sprintf(fmt_game_over_sub, i, arrRst[i]))
	}
	buf.Truncate(buf.Len() - 1)
	for _, p := range this.arrPlayers {
		if p != nil {
			p.SendMessage(fmt.Sprintf(fmt_game_over, buf.String()))
		}
	}
	this.ToBroadInfo()
}

func (this *DeskMnager) PutMessage(text string, site int) {
	msg := &DeskMsg{Text: text, Site: site}
	this.chDeskMsg <- msg
}

func (this *DeskMnager) ToBroadInfo() {
	this.chDeskBoard <- true
}

func (this *DeskMnager) Routine_Board() {
	Ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-this.chDeskBoard:
			this.broadDeskInfo()
		case <-Ticker.C:
			this.KickPlayer() // 长时间不开始，踢掉
		case msg := <-this.chDeskMsg:
			this.ProcessMsg(msg)
		}
	}
}

func (this *DeskMnager) ProcessMsg(*DeskMsg) {

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

// 长时间不开始，踢掉
func (this *DeskMnager) KickPlayer() {
	nowtime := time.Now().Unix()

	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()

	for k, v := range this.MapAddTimes {
		// 超过一分钟不准备
		p := this.arrPlayers[k]
		if p != nil && v != 0 && nowtime-v > 60 && p.Ready == 0 {
			p.SendMessage(fmt.Sprintf(fmt_timeout))
		}
	}
}
