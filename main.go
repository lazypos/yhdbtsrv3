package main

import (
	"./yhdbt"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flog, _ := os.OpenFile("yhdbt.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0x666)
	//log.SetOutput(flog)
	log.SetFlags(log.LstdFlags)
	defer flog.Close()

	log.Println(`[MAIN] server start! version: 2017-10-08`)
	if err := yhdbt.GDBOpt.Open(`./yhdbt_db`); err != nil {
		log.Println(`[MAIN] db open error:`, err)
		return
	}
	defer yhdbt.GDBOpt.Close()

	// yhdbt.ParseDB()
	// return

	yhdbt.GProcess.Init()
	yhdbt.GLogin.Start()
	yhdbt.GHall.Start()
	yhdbt.GKicked.Start()
	yhdbt.GWorker.Start()
	yhdbt.GTCPServer.Start("9998")
	yhdbt.GRegistServer.Start()
}
