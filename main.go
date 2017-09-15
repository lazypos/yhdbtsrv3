package main

import (
	"./yhdbt"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println(`[MAIN] server start!`)
	if err := yhdbt.GDBOpt.Open(`./yhdbt_db`); err != nil {
		log.Println(`[MAIN] db open error:`, err)
		return
	}
	defer yhdbt.GDBOpt.Close()

	yhdbt.GProcess.Init()
	yhdbt.GLogin.Start()
	yhdbt.GKicked.Start()
	yhdbt.GWorker.Start()
	yhdbt.GTCPServer.Start("5183")
	yhdbt.GRegistServer.Start()
}
