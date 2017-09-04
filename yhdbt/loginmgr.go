package yhdbt

import (
	"crypto/md5"
	"fmt"
	"net"
	"sync"
	"time"
)

type loginInfo struct {
	loginUid  string
	loginTime int64
}

type LoginManager struct {
	MapPlayers  map[string]*PlayerInfo //session->pinfp
	MapSession  map[string]string      //uid->session
	MapLoginKey map[string]*loginInfo  //loginkey->time
	muxLoginKey sync.Mutex
	muxPlayers  sync.Mutex
	loginPool   *sync.Pool
	playerPool  *sync.Pool
}

var GLogin = &LoginManager{}

func (this *LoginManager) Start() {
	this.loginPool = &sync.Pool{New: func() interface{} { return new(loginInfo) }}
	this.playerPool = &sync.Pool{New: func() interface{} { return new(PlayerInfo) }}
	this.MapPlayers = make(map[string]*PlayerInfo)
	this.MapSession = make(map[string]string)
	this.MapLoginKey = make(map[string]*loginInfo)
	go this.Routine_CheckTimeOut()
}

func (this *LoginManager) SaveLoginKey(key, uid string) {
	this.muxLoginKey.Lock()
	defer this.muxLoginKey.Unlock()

	lInfo := this.loginPool.Get().(*loginInfo)
	lInfo.loginUid = uid
	lInfo.loginTime = time.Now().Unix()
	this.MapLoginKey[key] = lInfo
}

func (this *LoginManager) Routine_CheckTimeOut() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			this.CheckLoginKeyTimeOut()
		}
	}
}

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

func (this *LoginManager) GetPlayerInfo(conn net.Conn, uid string) *PlayerInfo {
	this.muxPlayers.Lock()
	defer this.muxPlayers.Unlock()

	// 重复登陆,踢下线
	var pInfo *PlayerInfo = nil
	session, sok := this.MapSession[uid]
	if sok {
		pInfo = this.MapPlayers[uid]
		GKicked.AddTick(pInfo)
	}
	// 新session
	session = fmt.Sprintf(`%x`, md5.Sum([]byte(time.Now().String()+uid)))
	this.MapSession[uid] = session

	if pInfo == nil {
		pInfo := this.playerPool.Get().(*PlayerInfo)
		pInfo.Conn = conn
		pInfo.Uid = uid
		pInfo.NickName = string(GDBOpt.GetValue([]byte(fmt.Sprintf(`%s_nick`, uid)))[:])
		pInfo.Score = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_score`, uid)))
		pInfo.Win = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_win`, uid)))
		pInfo.Lose = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_lose`, uid)))
		pInfo.Run = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_run`, uid)))
		this.MapPlayers[uid] = pInfo
	}
	pInfo.LastOnline = time.Now().Unix()
	return pInfo
}

func (this *LoginManager) OnConnect(loginkey string) string {
	this.muxLoginKey.Lock()
	defer this.muxLoginKey.Unlock()
	lInfo, ok := this.MapLoginKey[loginkey]
	if ok && lInfo != nil {
		return lInfo.loginUid
	}
	return ""
}
