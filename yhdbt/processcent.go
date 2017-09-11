package yhdbt

import (
	"encoding/json"
	"fmt"
	"sync"
)

const (
	cmd_error         = 0x1000 //命令出错
	cmd_query_version = 0x1001 //查询版本
	cmd_add_desk      = 0x1002 //加入桌子
)

const (
	err_ok           = 0x3000
	err_desk_full    = 0x3001 //桌子满
	err_desk_noexist = 0x3002 //桌子不存在
)

// 回复信息
const (
	// 查询版本号
	fmt_query = `{"opt":"query","version":"46"}`
	// 创建桌子，成功桌号， 错误代码
	fmt_create_desk = `{"opt":"add","subopt":"create", "result":"%d"}`
	fmt_add         = `{"opt":"add","desk":"%d","site":"%d","name":"%s"}`
	fmt_change      = `{"opt":"change","info":[
					{"site":"0","name":"%s","ready":"%d","win":"%d","lose":"%d","run":"%d"},
					{"site":"1","name":"%s","ready":"%d","win":"%d","lose":"%d","run":"%d"},
					{"site":"2","name":"%s","ready":"%d","win":"%d","lose":"%d","run":"%d"},
					{"site":"3","name":"%s","ready":"%d","win":"%d","lose":"%d","run":"%d"}]}`
	fmt_start = `{"opt":"start","cards":"%s"}`
	fmt_run   = `{"opt":"run","site":"%d","name":"%s"}`
	fmt_over  = `{"opt":"over","info":[
                    {"site":"0","name":"%s","result":"%d"},
                    {"site":"1","name":"%s","result":"%d"},
                    {"site":"2","name":"%s","result":"%d"},
                    {"site":"3","name":"%s","result":"%d"}]}`
	fmt_game_put   = `{"opt":"game","per":"%d","cards":"%s","surplus":"%d","score":"%d","now":"%d","must":"%d"}`
	fmt_score      = `{"opt":"score","p0":"%d","p1":"%d"}`
	fmt_error      = `{"opt":"error"}`
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
	switch cmd {
	case cmd_query_version:
		return this.process_version(text, p)
	case cmd_add_desk:
		return this.process_add_desk(text, p)
	}
	return fmt.Errorf(`[PROCESS] unknow cmd`)
}

//查询版本
func (this *ProcessCent) process_version(text string, p *PlayerInfo) error {
	p.SendMessage(fmt_query)
	return nil
}

//请求加入桌子
func (this *ProcessCent) process_add_desk(text string, p *PlayerInfo) error {
	qm := this.msgPool.Get().(*QueryMessage)
	if err := json.Unmarshal([]byte(text), qm); err != nil {
		return fmt.Errorf(`[PROCESS] content error:`, err)
	}
	//创建桌子
	if qm.Opt == "create" {
		seq := GHall.AddDesk(-1, p)
		p.SendMessage(fmt.Sprintf(fmt_create_desk, seq))
	}
	//加入桌子
	if qm.Opt == "add" {
		seq := GHall.AddDesk(qm.DeskNum, p)
		p.SendMessage(fmt.Sprintf(fmt_create_desk, seq))
	}
	return nil
}
