package yhdbt

import (
	"bytes"
	"encoding/json"
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

type DeskInfo struct {
	Site  int32  `json:"site"`  //位号
	Type  string `json:"type"`  //类型
	Cards string `json:"cards"` //出牌
}

// 游戏相关
type DeskMnagerEx struct {
	baseScore   int   //基础分
	nLastPutSit int   //上一轮出牌玩家
	nNowPutSit  int   //当前出牌玩家
	nLastCards  []int //上一轮出的牌
	nDeskScore  int   //桌面分数
	nP0Score    int   //甲方得分
	nP1Score    int   //乙方得分
	RunCounts   int   //出完人数
	TimerCounts int   //检测超时断线
}

//桌子
type DeskMnager struct {
	DeskMnagerEx

	arrPlayers  []*PlayerInfo //玩家列表
	DeskNum     int           //桌好
	bPlaying    bool          //是否在游戏
	MapAddTimes map[int]int64 //玩家加入桌子的时间
	bCreate     bool          //是否是创建的

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
	this.bCreate = false
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
			p.SiteNum = i
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

	defer func() {
		p.SiteNum = -1
	}()

	empty := true
	if !this.bPlaying { // 如果游戏还没开始
		for i, v := range this.arrPlayers {
			if v != nil {
				if p.Uid == v.Uid {
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
	p.Zong += 1
	if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_zong`, p.Uid)), p.Zong); err != nil {
		log.Println(`[DESK] save player Zong error:`, p.Uid, p.Zong)
	}
	this.arrPlayers[p.SiteNum] = nil
	p.DeskNum = -1
	for _, v := range this.arrPlayers {
		if v != nil {
			v.Zong += 1
			if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_zong`, v.Uid)), v.Zong); err != nil {
				log.Println(`[DESK] save player Zong error:`, v.Uid, v.Zong)
			}
			v.SendMessage(fmt.Sprintf(fmt_run, p.SiteNum, p.NickName, this.baseScore*4))
		}
	}
	p.Score -= this.baseScore * 4
	if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_score`, p.Uid)), p.Score); err != nil {
		log.Println(`[DESK] save player score error:`, p.Uid, p.Score)
	}
	this.gameOver(true, []int{})
}

// 游戏结束
func (this *DeskMnager) gameOver(run bool, arrRst []int) {
	//更新状态
	this.bPlaying = false
	this.nLastPutSit = -1
	for i, p := range this.arrPlayers {
		this.MapAddTimes[i] = time.Now().Unix()
		if p != nil {
			p.Ready = 0
		}
	}
	if run {
		return
	}

	//保存成绩
	this.SaveResult(arrRst)

	//广播信息
	for i, p := range this.arrPlayers {
		if p != nil {
			p.SendMessage(fmt.Sprintf(fmt_game_over, arrRst[i]))
		}
	}
	this.ToBroadInfo()
}

func (this *DeskMnager) SaveResult(rst []int) {
	for i, v := range rst {
		//计算盘和分数
		p := this.arrPlayers[i]
		if v > 0 {
			p.Score += (v * this.baseScore * 9) / 10
		} else {
			p.Score += v * this.baseScore
		}
		if v > 0 {
			p.Win += v
			if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_win`, p.Uid)), p.Win); err != nil {
				log.Println(`[DESK] save player win error:`, p.Uid, p.Win)
			}
		} else if v < 0 {
			p.Lose -= v
			if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_lose`, p.Uid)), p.Lose); err != nil {
				log.Println(`[DESK] save player lose error:`, p.Uid, p.Lose)
			}
		} else {
			p.He += 1
			if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_he`, p.Uid)), p.He); err != nil {
				log.Println(`[DESK] save player he error:`, p.Uid, p.He)
			}
		}
		if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_score`, p.Uid)), p.Score); err != nil {
			log.Println(`[DESK] save player score error:`, p.Uid, p.Score)
		}
		p.Zong += 1
		if err := GDBOpt.PutValueInt([]byte(fmt.Sprintf(`%s_zong`, p.Uid)), p.Zong); err != nil {
			log.Println(`[DESK] save player Zong error:`, p.Uid, p.Zong)
		}
	}
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

func (this *DeskMnager) ProcessMsg(m *DeskMsg) {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()

	if !this.bPlaying {
		return
	}

	deskmsg := &DeskInfo{}
	if err := json.Unmarshal([]byte(m.Text), deskmsg); err != nil {
		log.Println(`[DESK] content error:`, err)
		this.playerRun(this.arrPlayers[m.Site])
		return
	}

	must := 0
	next := m.Site
	putCards := StringTointArr(deskmsg.Cards)
	if len(putCards) == 0 { // 玩家没牌
		for i := 0; i < 4; i++ {
			// 看下一家谁出
			next = this.GetNextPut(next)
			// 大了一轮，得分
			if next == this.nLastPutSit {
				must = 1
				this.nLastCards = putCards
				this.CaluScore()
				//如果出完了，对家出
				if this.arrPlayers[next].RunNum != -1 {
					next = (next + 2) % 4
					break
				}
			}
			// 下一家还没出完的出牌
			if this.arrPlayers[next].RunNum == -1 {
				break
			}
		}
	} else {
		// 出牌错误, 强制踢掉
		if !IsBigger(this.nLastCards, putCards) {
			log.Println(`[DESK] put cards error, litter than per.`)
			this.playerRun(this.arrPlayers[m.Site])
			return
		}
		score, err := this.arrPlayers[m.Site].PutCards(putCards)
		if err != nil {
			log.Println(`[DESK] put cards error, 手牌不符合.`)
			this.playerRun(this.arrPlayers[m.Site])
			return
		}

		this.nLastPutSit = next
		//出完牌
		if len(this.arrPlayers[m.Site].ArrCards) == 0 {
			this.arrPlayers[m.Site].RunNum = this.RunCounts
			this.RunCounts++
			log.Println(`[DESK]`, m.Site, "over")
		}
		this.nDeskScore += score
		for i := 0; i < 4; i++ {
			next = this.GetNextPut(next)
			//log.Println(next, this.arrPlayers[next].RunNum )
			if this.arrPlayers[next].RunNum == -1 {
				break
			}
		}
		this.nLastCards = putCards
	}

	this.nNowPutSit = next
	this.TimerCounts = 0
	for _, p := range this.arrPlayers {
		p.SendMessage(fmt.Sprintf(fmt_game_put, m.Site, deskmsg.Cards, len(this.arrPlayers[m.Site].ArrCards), this.nDeskScore, next, must))
		p.SendMessage(fmt.Sprintf(fmt_score, this.nP0Score, this.nP1Score))
	}
	// 计算是否结束
	run := []int{-1, -1, -1, -1}
	for i := 0; i < 4; i++ {
		run[i] = this.arrPlayers[i].RunNum
	}
	if over, arrRst := IsOver(this.nP0Score, this.nP1Score, run); over {
		this.gameOver(false, arrRst)
	}
}

func (this *DeskMnager) GetNextPut(site int) int {
	for i := 1; i < 4; i++ {
		next := (site + i) % 4
		return next
	}
	return -1
}

func (this *DeskMnager) CaluScore() {
	this.nLastCards = []int{}

	if this.nLastPutSit%2 == 0 {
		this.nP0Score += this.nDeskScore
	} else {
		this.nP1Score += this.nDeskScore
	}
	this.nDeskScore = 0
}

//广播桌子信息
func (this *DeskMnager) broadDeskInfo() {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()

	buf := bytes.NewBufferString("")
	for i, p := range this.arrPlayers {
		if p != nil {
			buf.WriteString(fmt.Sprintf(fmt_change_sub, i, p.NickName, p.Ready, p.Score, p.Win, p.Lose, p.Run, p.Sex, p.He, p.Zong))
		} else {
			buf.WriteString(fmt.Sprintf(fmt_change_sub, i, "", 0, 0, 0, 0, 0, 0, 0, 0))
		}
	}
	buf.Truncate(buf.Len() - 1)
	for _, p := range this.arrPlayers {
		if p != nil {
			p.SendMessage(fmt.Sprintf(fmt_change, buf.String()))
			//log.Println(fmt.Sprintf(fmt_change, buf.String()))
		}
	}
}

// 长时间不开始，踢掉
func (this *DeskMnager) KickPlayer() {
	nowtime := time.Now().Unix()

	this.muxDesk.Lock()
	for k, v := range this.MapAddTimes {
		// 超过一分钟不准备
		p := this.arrPlayers[k]
		if p != nil && v != 0 && nowtime-v > 60 && p.Ready == 0 && !this.bPlaying {
			p.SendMessage(fmt.Sprintf(fmt_timeout))
			this.arrPlayers[k] = nil
			this.MapAddTimes[k] = 0
			//GProcess.ProcessCmd(cmd_desk_leave, "", p)
		}
	}
	this.muxDesk.Unlock()

	this.TimerCounts += 5
	if this.TimerCounts > 35 && this.bPlaying {
		this.LeavePlayer(this.arrPlayers[this.nNowPutSit])
		this.bPlaying = false
	}
}

// 游戏开始
func (this *DeskMnager) GmeStart(n int) {
	// 初始化
	this.nLastPutSit = -1
	this.nLastCards = []int{}
	this.bPlaying = true
	this.nDeskScore = 0
	this.nP0Score = 0
	this.nP1Score = 0
	this.RunCounts = 0
	this.TimerCounts = 0
	for i, _ := range this.arrPlayers {
		this.arrPlayers[i].RunNum = -1
		this.arrPlayers[i].Ready = 0
	}

	//随机调整位置
	p0 := GRand.Intn(3)
	p1 := GRand.Intn(3)
	if p0 != p1 {
		log.Println("调整座位", p0, p1)
		this.arrPlayers[p0], this.arrPlayers[p1] = this.arrPlayers[p1], this.arrPlayers[p0]
		this.arrPlayers[p0].SiteNum = p0
		this.arrPlayers[p1].SiteNum = p1
		for _, p := range this.arrPlayers {
			p.SendMessage(fmt.Sprintf(fmt_site_change, p0, p1))
		}
		this.ToBroadInfo()
	}

	// 发牌
	arrCards, arrCardsint := Create4Cards()
	for i, p := range this.arrPlayers {
		this.arrPlayers[i].ArrCards = arrCardsint[i]
		//log.Println(this.arrPlayers[i].ArrCards)
		p.SendMessage(fmt.Sprintf(fmt_start, arrCards[i], n))
	}
	put := GRand.Intn(3)
	this.nLastPutSit = put
	this.nNowPutSit = put
	for _, p := range this.arrPlayers {
		p.SendMessage(fmt.Sprintf(fmt_game_put, -1, "", 54, 0, put, 1))
	}
}

func (this *DeskMnager) OnReady() {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()
	if this.bPlaying {
		return
	}
	// 是否全准备
	lowscore := 500 //最低分
	var allReady = true
	for i := 0; i < 4; i++ {
		p := this.arrPlayers[i]
		if p == nil || p.Ready == 0 {
			allReady = false
			break
		}
		if p != nil {
			if p.Score < lowscore {
				lowscore = p.Score
			}
		}
	}
	if !allReady {
		this.ToBroadInfo()
		return
	}
	//算底分
	this.baseScore = 10 + (lowscore/200)*10
	log.Println("本局底分：", this.baseScore, this.DeskNum)
	this.GmeStart(this.baseScore)
}

func (this *DeskMnager) Empty() bool {
	this.muxDesk.Lock()
	defer this.muxDesk.Unlock()

	for _, v := range this.arrPlayers {
		if v != nil {
			return false
		}
	}
	return true
}
