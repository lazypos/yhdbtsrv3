package yhdbt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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
	cmd_query_online  = 0x1008 //查询在线人数
	cmd_add_hall      = 0x1009 //加入大厅
	cmd_query_self    = 0x1010 //查询自己信息
	cmd_exit          = 0x1011 //退出
)

const (
	err_desk_full    = -0x3001 //桌子满
	err_desk_noexist = -0x3002 //桌子不存在
)

// 回复信息
const (
	// 查询版本号
	fmt_query_version = `{"opt":"version","version":"65"}`
	// 查询在线, 返回在线人数和在玩的桌子数
	fmt_query_online = `{"opt":"online","count":"%d","desk":"%d"}`
	// 查询排行榜 昵称和分数
	fmt_query_rank     = `{"opt":"rank","info":[%s]}`
	fmt_query_rank_sub = `{"id":"%d","nick":"%s","score":"%s"},`
	//{"id":"0","nick":"%s","score":"%d"},
	//{"id":"1","nick":"%s","score":"%d"},
	//{"id":"2","nick":"%s","score":"%d"},
	//{"id":"3","nick":"%s","score":"%d"},
	//{"id":"4","nick":"%s","score":"%d"}
	// 加入桌子，成功桌号/座位， 错误代码
	fmt_add_desk = `{"opt":"add","desk":"%d","site":"%d"}`
	// 桌上玩家变更
	fmt_change     = `{"opt":"change","info":[%s]}`
	fmt_change_sub = `{"site":"%d","name":"%s","ready":"%d","score":"%d","win":"%d","lose":"%d","run":"%d","sex":"%d","he":"%d","zong":"%d"},`
	//{"site":"0","name":"%s","ready":"%d","score":"%d","win":"%d","lose":"%d","run":"%d"},
	//{"site":"1","name":"%s","ready":"%d","score":"%d","win":"%d","lose":"%d","run":"%d"},
	//{"site":"2","name":"%s","ready":"%d","score":"%d","win":"%d","lose":"%d","run":"%d"},
	//{"site":"3","name":"%s","ready":"%d","score":"%d","win":"%d","lose":"%d","run":"%d"}
	// 游戏开始
	fmt_start = `{"opt":"start","cards":"%s","base":"%d"}`
	// 玩家逃跑 扣多少分
	fmt_run = `{"opt":"run","site":"%d","name":"%s","score":"%d"}`
	// 游戏结束
	fmt_game_over = `{"opt":"over","result":"%d"}`
	// fmt_game_over_sub = `{"site":"%d","result":"%d"},`
	// //{"site":"0","name":"%s","result":"%d"},
	// //{"site":"1","name":"%s","result":"%d"},
	// //{"site":"2","name":"%s","result":"%d"},
	// //{"site":"3","name":"%s","result":"%d"}
	// 玩家出牌 前一家出牌，剩余，桌面分数，现在出牌，是否必须出
	fmt_game_put = `{"opt":"game","per":"%d","cards":"%s","surplus":"%d","score":"%d","now":"%d","must":"%d"}`
	// 广播桌子当前两队得分
	fmt_score = `{"opt":"score","p0":"%d","p1":"%d"}`
	// 数据异常
	fmt_error = `{"opt":"error"}`
	// 玩家进入桌子长时间不准备 <- 玩家收到后发送离开桌子的请求
	fmt_timeout = `{"opt":"timeout"}`
	// 被踢下线
	fmt_kicked = `{"opt":"kicked"}`
	// 玩家登陆的时候返回玩家信息
	fmt_plyer_info = `{"opt":"login","nick":"%s","score":"%d","win":"%d","lose":"%d","run":"%d","sex":"%d","he":"%d","zong":"%d"}`
	// 积分不够
	fmt_score_less = `{"opt":"less"}`
	// 开局前位置变换
	fmt_site_change = `{"opt":"site","p0":"%d","p1":"%d"}`
)

type QueryMessage struct {
	Opt     string `json:"opt"`     //操作
	DeskNum int    `json:"desk"`    //桌号
	Site    int    `json:"site"`    //位号
	Type    string `json:"type"`    //类型
	Cards   string `json:"cards"`   //出牌
	Message string `json:"message"` //其他
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
	// if cmd != cmd_query_online && cmd != cmd_heart {
	// 	log.Println(`[PROCESS] recv cmd:`, cmd, string(text[:]))
	// }

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
	case cmd_query_online:
		return this.process_online(p)
	case cmd_query_self:
		return this.prcess_query_self(p)
	default:
		return this.prcess_error(p)
	}
	log.Println(`[PROCESS] unknow cmd`, cmd)
	return fmt.Errorf(`[PROCESS] unknow cmd %v`, cmd)
}

func (this *ProcessCent) prcess_query_self(p *PlayerInfo) error {
	p.SendMessage(fmt.Sprintf(fmt_plyer_info, p.NickName, p.Score, p.Win, p.Lose, p.Run, p.Sex, p.He, p.Zong))
	return nil
}

// 数据异常
func (this *ProcessCent) prcess_error(p *PlayerInfo) error {
	this.process_leave("", p)
	//p.SendMessage(fmt_error)
	return nil
}

//在线人数
func (this *ProcessCent) process_online(p *PlayerInfo) error {
	n, d := GHall.QueryPlayerCounts()
	p.SendMessage(fmt.Sprintf(fmt_query_online, n, d))
	return nil
}

//查询版本
func (this *ProcessCent) process_version(text string, p *PlayerInfo) error {
	p.SendMessage(fmt_query_version)
	return nil
}

//查询排名
func (this *ProcessCent) process_rank(text string, p *PlayerInfo) error {
	m := GWorker.GetScoreRank()
	buf := bytes.NewBufferString("")
	for k, v := range m {
		if v != nil {
			buf.WriteString(fmt.Sprintf(fmt_query_rank_sub, k, v.nick, v.socre))
		}
	}
	if buf.Len() > 0 {
		buf.Truncate(buf.Len() - 1)
	}
	p.SendMessage(fmt.Sprintf(fmt_query_rank, buf.String()))
	return nil
}

//请求加入桌子
func (this *ProcessCent) process_add_desk(text string, p *PlayerInfo) error {
	qm := this.msgPool.Get().(*QueryMessage)
	defer this.msgPool.Put(qm)
	if err := json.Unmarshal([]byte(text), qm); err != nil {
		return fmt.Errorf(`[PROCESS] content error: %v`, err)
	}
	//创建桌子
	var desknum = qm.DeskNum
	var sitenum = 0
	var desk *DeskMnager = nil
	if qm.Opt == "create" {
		desk, sitenum = GHall.CreateDesk(p)
		if desk == nil {
			p.SendMessage(fmt.Sprintf(fmt_add_desk, -1, sitenum))
			return nil
		}
	} else {
		//加入桌子
		if desknum > 0 {
			desk, sitenum = GHall.AddDesk(desknum, p)
			if desk == nil {
				p.SendMessage(fmt.Sprintf(fmt_add_desk, -1, sitenum))
				return nil
			}
		} else { //快速加入
			desk, sitenum = GHall.FastAddDesk(p)
			if desk == nil {
				p.SendMessage(fmt.Sprintf(fmt_add_desk, -1, sitenum))
				return nil
			}
		}
	}
	log.Println(`加入桌子成功`, desk.DeskNum, sitenum)
	p.SendMessage(fmt.Sprintf(fmt_add_desk, desk.DeskNum, sitenum))
	//广播消息
	p.DeskNum = desk.DeskNum
	p.SiteNum = sitenum
	p.Ready = 0
	GHall.BroadDeskInfo(desk.DeskNum)
	return nil
}

//玩家举手
func (this *ProcessCent) process_ready(text string, p *PlayerInfo) error {
	if p.Score < 50 {
		p.SendMessage(fmt_score_less)
		return nil
	}
	p.Ready = 1
	desk := GHall.GetDesk(p.DeskNum)
	if desk != nil {
		desk.OnReady()
		return nil
	}
	return fmt.Errorf(`[PROCESS] desk error: no desk %v`, p.DeskNum)
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
