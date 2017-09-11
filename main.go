package main

import (
	"./yhdbt"
	"log"
)

func main() {

	m := make(map[int]int)
	m[2] = 1
	m[3] = 2
	log.Println(len(m))
	delete(m, 3)
	log.Println(len(m))
	return

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
	yhdbt.GTCPServer.Start("5183")
	yhdbt.GRegistServer.Start()
}
