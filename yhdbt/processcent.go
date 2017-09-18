package yhdbt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// 请求命令
const (
	cmd_error         = 0x1000 //命令出错
	cmd_query_version = 0x1001 //查询版本
	cmd_add_desk      = 0x1002 //加入桌子
	cmd_query_rank    = 0x1003 //查询排行
	cmd_desk_ready    = 0x1004 //举手准备
	cmd_desk_leave    = 0x1005 //离开桌子
	cmd_heart         = 0x1006 //心跳
	cmd_put_cards     = 0x1007 //玩家出牌
)

const (
	err_desk_full    = -0x3001 //桌子满
	err_desk_noexist = -0x3002 //桌子不存在
)

// 回复信息
const (
	// 查询版本号
	fmt_query = `{"opt":"query","version":"46"}`
	// 查询排行榜 昵称和分数
	fmt_query_rank     = `{"opt":"rank","info":[%s]}`
	fmt_query_rank_sub = `{"id":"%d","nick":"%s","score":"%d"},`
	//{"id":"0","nick":"%s","score":"%d"},
	//{"id":"1","nick":"%s","score":"%d"},
	//{"id":"2","nick":"%s","score":"%d"},
	//{"id":"3","nick":"%s","score":"%d"},
	//{"id":"4","nick":"%s","score":"%d"},
	//{"id":"5","nick":"%s","score":"%d"},
	//{"id":"6","nick":"%s","score":"%d"},
	//{"id":"7","nick":"%s","score":"%d"},
	//{"id":"8","nick":"%s","score":"%d"},
	//{"id":"9","nick":"%s","score":"%d"}
	// 加入桌子，成功桌号/座位， 错误代码
	fmt_add_desk = `{"opt":"add","desk":"%d","site":"%d"}`
	// 桌上玩家变更
	fmt_change     = `{"opt":"change","info":[%s]}`
	fmt_change_sub = `{"site":"%d","name":"%s","ready":"%d","socre":"%d","win":"%d","lose":"%d","run":"%d"},`
	//{"site":"0","name":"%s","ready":"%d","socre":"%d","win":"%d","lose":"%d","run":"%d"},
	//{"site":"1","name":"%s","ready":"%d","socre":"%d","win":"%d","lose":"%d","run":"%d"},
	//{"site":"2","name":"%s","ready":"%d","socre":"%d","win":"%d","lose":"%d","run":"%d"},
	//{"site":"3","name":"%s","ready":"%d","socre":"%d","win":"%d","lose":"%d","run":"%d"}
	// 游戏开始
	fmt_start = `{"opt":"start","cards":"%s"}`
	// 玩家逃跑 扣多少分
	fmt_run = `{"opt":"run","site":"%d","name":"%s","score":"%d"}`
	// 游戏结束
	fmt_game_over     = `{"opt":"over","info":[%s]}`
	fmt_game_over_sub = `{"site":"%d","result":"%d"},`
	//{"site":"0","name":"%s","result":"%d"},
	//{"site":"1","name":"%s","result":"%d"},
	//{"site":"2","name":"%s","result":"%d"},
	//{"site":"3","name":"%s","result":"%d"}
	// 玩家出牌 前一家出牌，剩余，桌面分数，现在出牌，是否必须出
	fmt_game_put = `{"opt":"game","per":"%d","cards":"%s","surplus":"%d","score":"%d","now":"%d","must":"%d"}`
	fmt_score    = `{"opt":"score","p0":"%d","p1":"%d"}`
	fmt_error    = `{"opt":"error"}`
	// 玩家进入桌子长时间不准备 <- 玩家收到后发送离开桌子的请求
	fmt_timeout    = `{"opt":"timeout"}`
	fmt_playerinfo = `{"opt":"","win0":"","lose0":"","run0":""}`
)

type QueryMessage struct {
	Opt     string `json:"opt"`   //操作
	DeskNum int    `json:"desk"`  //桌号
	Site    int    `json:"site"`  //位号
	Type    string `json:"type"`  //类型
	Cards   string `json:"cards"` //出牌
	Key     string `json:"key"`   //用户ID
}

// 命令处理
type ProcessCent struct {
	msgPool *sync.Pool
}

var GProcess = &ProcessCent{}

func (this *ProcessCent) Init() {
	this.msgPool = &sync.Pool{New: func() interface{} { return new(QueryMessage) }}
}

//命令处理函数
func (this *ProcessCent) ProcessCmd(cmd int, text string, p *PlayerInfo) error {
	//更新在线时间
	p.LastOnline = time.Now().Unix()

	switch cmd {
	case cmd_query_version:
		return this.process_version(text, p)
	case cmd_add_desk:
		return this.process_add_desk(text, p)
	case cmd_query_rank:
		return this.process_rank(text, p)
	case cmd_desk_ready:
		return this.process_ready(text, p)
	case cmd_desk_leave:
		return this.process_leave(text, p)
	case cmd_heart:
		return this.process_heart(text, p)
	case cmd_put_cards:
		return this.process_put_cards(text, p)
	}
	return fmt.Errorf(`[PROCESS] unknow cmd`)
}

//查询版本
func (this *ProcessCent) process_version(text string, p *PlayerInfo) error {
	p.SendMessage(fmt_query)
	return nil
}

//查询排名
func (this *ProcessCent) process_rank(text string, p *PlayerInfo) error {
	m := GWorker.GetScoreRank()
	buf := bytes.NewBufferString("")
	for k, v := range m {
		buf.WriteString(fmt.Sprintf(fmt_query_rank_sub, k, v.nick, v.socre))
	}
	buf.Truncate(buf.Len() - 1)
	p.SendMessage(fmt.Sprintf(fmt_query_rank, buf.String()))
	return nil
}

//请求加入桌子
func (this *ProcessCent) process_add_desk(text string, p *PlayerInfo) error {
	qm := this.msgPool.Get().(*QueryMessage)
	defer this.msgPool.Put(qm)
	if err := json.Unmarshal([]byte(text), qm); err != nil {
		return fmt.Errorf(`[PROCESS] content error:`, err)
	}
	//创建桌子
	var desknum = qm.DeskNum
	if qm.Opt == "create" {
		desknum = -1
	}

	desk, sid := GHall.AddDesk(desknum, p)
	if desk == nil {
		p.SendMessage(fmt.Sprintf(fmt_add_desk, desk.DeskNum, sid))
		return nil
	}
	p.SendMessage(fmt.Sprintf(fmt_add_desk, err_desk_full, sid))
	//广播消息
	p.DeskNum = desk.DeskNum
	p.SiteNum = sid
	GHall.BroadDeskInfo(desk.DeskNum)
	return nil
}

//玩家举手
func (this *ProcessCent) process_ready(text string, p *PlayerInfo) error {
	p.Ready = 1
	GHall.BroadDeskInfo(p.DeskNum)
	return nil
}

//离开桌子
func (this *ProcessCent) process_leave(text string, p *PlayerInfo) error {
	deskNum := p.DeskNum
	GHall.LeaveDesk(p)
	GHall.BroadDeskInfo(deskNum)
	return nil
}

//心跳
func (this *ProcessCent) process_heart(text string, p *PlayerInfo) error {
	p.LastOnline = time.Now().Unix()
	return nil
}

//出牌
func (this *ProcessCent) process_put_cards(text string, p *PlayerInfo) error {
	desk := GHall.GetDesk(p.DeskNum)
	if desk != nil {
		desk.PutMessage(text, p.SiteNum)
		return nil
	}
	return fmt.Errorf(`[PROCESS] desk error: no desk`, p.DeskNum)
}
