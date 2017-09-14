package yhdbt

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Msg struct {
	msg      string
	callback func(string)
}

type PlayerInfoEx struct {
	chBroad   chan *Msg
	msgPool   *sync.Pool
	muxPlayer sync.Mutex
	ArrCards  []int //手牌

	Ready   int
	DeskNum int
	SiteNum int
}

//玩家
type PlayerInfo struct {
	PlayerInfoEx
	Session    string   //通讯密码
	NickName   string   //昵称
	Uid        string   //用户id
	Score      int      //积分
	Win        int      //胜利计数
	Lose       int      //失败计数
	Run        int      //逃跑计数
	LoginTimes int      //登陆计数
	LastOnline int64    //最后在线时间
	Conn       net.Conn //连接句柄
}

func (this *PlayerInfo) Init() {
	this.chBroad = make(chan *Msg, 10)
	this.msgPool = &sync.Pool{New: func() interface{} { return new(Msg) }}
	go this.Routine_Broad()
	go this.Routine_Recv()
}

//向玩家发送信息函数
func (this *PlayerInfo) SendMessageCB(content string, cb func(string)) {
	msg := this.msgPool.Get().(*Msg)
	msg.callback = cb
	msg.msg = content
	this.chBroad <- msg
}

func (this *PlayerInfo) SendMessage(content string) {
	this.SendMessageCB(content, nil)
}

//发送信息线程
func (this *PlayerInfo) Routine_Broad() {
	for {
		select {
		case msg := <-this.chBroad:
			log.Println(`[PLAYER] send message`, msg)
			if err := SendCommond(this.Conn, []byte(msg.msg)); err != nil {
				log.Println(`[PLAYER] send error`, err)
				GProcess.ProcessCmd(cmd_error, "", this)
			}
			this.msgPool.Put(msg)
		}
	}
}

//接受玩家信息线程
func (this *PlayerInfo) Routine_Recv() {
	type CommInfo struct {
		Cmd     int    `json:"cmd"`
		Session string `json:"cookie"`
		Text    string `json:"text"`
	}
	CInfo := &CommInfo{}
	for {
		content, err := RecvCommond(this.Conn)
		if err != nil {
			log.Println(`[PLAYER] recv error:`, err)
			break
		}

		if err = json.Unmarshal(content, CInfo); err != nil {
			log.Println(`[PLAYER] recv json error:`, err)
			break
		}

		this.muxPlayer.Lock()
		tmpSession := this.Session
		this.muxPlayer.Unlock()
		if CInfo.Session != tmpSession {
			break
		}
		// 处理请求
		log.Panicln(`[PLYAER] recv cmd:`, CInfo.Cmd, CInfo.Text)
		if err = GProcess.ProcessCmd(CInfo.Cmd, CInfo.Text, this); err != nil {
			log.Println(`[PLAYER] process cmd error,`, err)
			break
		}
	}
	// 处理断开连接
	GProcess.ProcessCmd(cmd_error, "", this)
}

//重新初始化玩家信息(可复用)
func (this *PlayerInfo) ReInit(conn net.Conn) {
	this.muxPlayer.Lock()
	defer this.muxPlayer.Unlock()
	this.Conn = conn
	this.Session = fmt.Sprintf(`%x`, md5.Sum([]byte(time.Now().String()+this.Uid)))
	this.LastOnline = time.Now().Unix()
}
