package main

import (
	"./yhdbt"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println(`[MAIN] server start! version: 2017-10-08`)
	if err := yhdbt.GDBOpt.Open(`./yhdbt_db`); err != nil {
		log.Println(`[MAIN] db open error:`, err)
		return
	}
	defer yhdbt.GDBOpt.Close()

	yhdbt.GProcess.Init()
	yhdbt.GLogin.Start()
	yhdbt.GHall.Start()
	yhdbt.GKicked.Start()
	yhdbt.GWorker.Start()
	yhdbt.GTCPServer.Start("9999")
	yhdbt.GRegistServer.Start()
}
