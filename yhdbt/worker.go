package yhdbt

import (
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
	go this.Routine_worker()
}

// 间隔更新线程
func (this *Worker) Routine_worker() {
	rankTicker := time.NewTicker(time.Hour)

	for {
		select {
		case <-rankTicker.C:
			this.UpdateRank()
		}
	}
}

// 更新排名
func (this *Worker) UpdateRank() {

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
