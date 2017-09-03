package yhdbt

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

type LoginManager struct {
	MapPlayers  map[string]*PlayerInfo //session->pinfp
	MapSession  map[string]string      //uid->session
	MapLoginKey map[string]int64       //loginkey->time
	muxLoginKey sync.Mutex
	muxPlayers  sync.Mutex
	PlayerPool  *sync.Pool
	chOffline   chan string
}

var GLogin = &LoginManager{}

func (this *LoginManager) Start() {
	this.PlayerPool = &sync.Pool{New: func() interface{} { return new(PlayerInfo) }}
	this.MapPlayers = make(map[string]*PlayerInfo)
	this.MapSession = make(map[string]string)
	this.MapLoginKey = make(map[string]int64)
	this.chOffline = make(chan string, 100)
	go this.Routine_CheckLoginKeyTimeOut()
}

func (this *LoginManager) SaveLoginKey(key string) {
	this.muxLoginKey.Lock()
	defer this.muxLoginKey.Unlock()
	this.MapLoginKey[loginKey] = time.Now().Unix()
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
		if now-v >= 60 {
			delete(this.MapLoginKey, k)
		}
	}
}

func (this *LoginManager) GetPlayerInfo(uid string) *PlayerInfo {
	this.muxPlayers.Lock()
	defer this.muxPlayers.Unlock()

	// 重复登陆,踢下线
	pInfo, pok := this.MapPlayers[uid]
	session, sok := this.MapSession[uid]
	if sok {
		this.chOffline <- session
		GKicked.AddTick(pInfo.Conn)
	}
	// 新session
	session = fmt.Sprintf(`%x`, md5.Sum([]byte(time.Now()+uid)))
	this.MapSession[uid] = session

	if !pok || pInfo == nil {
		pInfo = &PlayerInfo{}
		pInfo.Uid = uid
		pInfo.NickName = GDBOpt.GetValue([]byte(fmt.Sprintf(`%s_nick`, uid)))
		pInfo.Score = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_score`, uid)))
		pInfo.Win = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_win`, uid)))
		pInfo.Lose = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_lose`, uid)))
		pInfo.Run = GDBOpt.GetValueAsInt([]byte(fmt.Sprintf(`%s_run`, uid)))
		this.MapPlayers[uid] = pInfo
	}
	pInfo.LastOnline = time.Now().Unix()
	return pInfo
}
