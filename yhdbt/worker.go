package yhdbt

import (
	"log"
	"sync"
	"time"
)

type RankScoreInfo struct {
	nick  string
	socre string
}

// 间隔工作
type Worker struct {
	arrScore []*RankScoreInfo
	muxRank  sync.Mutex
}

var GWorker = &Worker{}

func (this *Worker) Start() {
	this.UpdateRank()
	go this.Routine_worker()
}

// 间隔更新线程
func (this *Worker) Routine_worker() {
	rankTicker := time.NewTicker(time.Hour * 1)

	for {
		select {
		case <-rankTicker.C:
			this.UpdateRank()
		}
	}
}

// 更新排名
func (this *Worker) UpdateRank() {
	// 遍历所有人员
	r, err := GDBOpt.GetMaxScore(5)
	if err != nil {
		log.Println(`获取排名信息失败.`)
		return
	}

	log.Println(`获取排名信息`, r)
	this.muxRank.Lock()
	defer this.muxRank.Unlock()
	this.arrScore = r
}

// 获取排名信息
func (this *Worker) GetScoreRank() map[int]*RankScoreInfo {
	m := make(map[int]*RankScoreInfo)
	this.muxRank.Lock()
	defer this.muxRank.Unlock()
	for i, v := range this.arrScore {
		m[i] = v
	}
	return m
}
