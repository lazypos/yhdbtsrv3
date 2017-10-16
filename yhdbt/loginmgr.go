package yhdbt

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type loginInfo struct {
	loginUid  string
	loginTime int64
}

//登陆管理
type LoginManager struct {
	MapPlayers  map[string]*PlayerInfo //session->pinfp
	MapLoginKey map[string]*loginInfo  //loginkey->time
	muxLoginKey sync.Mutex
	muxPlayers  sync.Mutex
	loginPool   *sync.Pool
	//playerPool  *sync.Pool
}

var GLogin = &LoginManager{}

func (this *LoginManager) Start() {
	this.loginPool = &sync.Pool{New: func() interface{} { return new(loginInfo) }}
	//this.playerPool = &sync.Pool{New: func() interface{} { return new(PlayerInfo) }}
	this.MapPlayers = make(map[string]*PlayerInfo)
	this.MapLoginKey = make(map[string]*loginInfo)
	go this.Routine_CheckTimeOut()
}

//登陆后保存登陆凭证
func (this *LoginManager) SaveLoginKey(key, uid string) {
	this.muxLoginKey.Lock()
	defer this.muxLoginKey.Unlock()

	lInfo := this.loginPool.Get().(*loginInfo)
	lInfo.loginUid = uid
	lInfo.loginTime = time.Now().Unix()
	this.MapLoginKey[key] = lInfo
}

//定期清除超时的登陆凭证
func (this *LoginManager) Routine_CheckTimeOut() {
	ticker := time.NewTicker(time.Second * 5)
	//tickerLeave := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			this.CheckLoginKeyTimeOut()
			// case <-tickerLeave.C:
			// 	this.CheckLineOff()
		}
	}
}

//处理掉线
func (this *LoginManager) CheckLineOff() {
	now := time.Now().Unix()

	this.muxPlayers.Lock()
	defer this.muxPlayers.Unlock()
	for _, v := range this.MapPlayers {
		if now-v.LastOnline > 120 {
			GProcess.ProcessCmd(cmd_error, "", v)
		}
	}
}

//清除凭证函数
func (this *LoginManager) CheckLoginKeyTimeOut() {
	now := time.Now().Unix()
	this.muxLoginKey.Lock()
	defer this.muxLoginKey.Unlock()
	for k, v := range this.MapLoginKey {
		if now-v.loginTime >= 60 {
			delete(this.MapLoginKey, k)
			this.loginPool.Put(v)
		}
	}
}

//登陆后，获取玩家的信息，如果玩家已经登陆，则从缓存中获取，否则从数据库查询
func (this *LoginManager) GetPlayerInfo(conn net.Conn, uid string) *PlayerInfo {
	this.muxPlayers.Lock()
	defer this.muxPlayers.Unlock()

	// 重复登陆,踢下线
	pold, ok := this.MapPlayers[uid]
	if ok && len(pold.Session) > 0 {
		log.Println(`[login] kicked.`)
		pold.Kicked()
	}
	// 如果是新登陆的
	pInfo := &PlayerInfo{}
	this.MapPlayers[uid] = pInfo
	pInfo.Init(conn)

	pInfo.Uid = uid
	pInfo.NickName = string(GDBOpt.GetValue([]byte(fmt.Sprintf(`%s_nick`, uid)))[:])
	pInfo.Score = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_score`, uid)))
	pInfo.Win = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_win`, uid)))
	pInfo.Lose = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_lose`, uid)))
	pInfo.Run = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_run`, uid)))
	pInfo.Sex = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_sex`, uid)))
	pInfo.He = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_he`, uid)))
	pInfo.Zong = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_zong`, uid)))

	return pInfo
}

//根据登陆凭证再次连接
func (this *LoginManager) OnConnect(loginkey string) string {
	this.muxLoginKey.Lock()
	defer this.muxLoginKey.Unlock()
	lInfo, ok := this.MapLoginKey[loginkey]
	if ok && lInfo != nil {
		return lInfo.loginUid
	}
	return ""
}
