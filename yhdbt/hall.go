package yhdbt

type HallManger struct {
	Desknum   int
	Playernum int
}

func (this *HallManger) Start() {
	this.Desknum = 0
	this.Playernum = 0
}
