package main

import (
	"./yhdbt"
	"log"
)

func main() {
	m := make(map[int]int)
	m[1] = 1
	m[2] = 2
	for _, v := range m {
		v = 3
		log.Println(v)
	}
	log.Println(m)
	return

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println(`[MAIN] server start!`)
	if err := yhdbt.GDBOpt.Open(`./yhdbt_db`); err != nil {
		log.Println(`[MAIN] db open error:`, err)
		return
	}
	defer yhdbt.GDBOpt.Close()

	//log.Println(yhdbt.CheckNickName("dajds1231231"))
	// s := fmt.Sprintf("%x", md5.Sum([]byte("123132132")))
	// log.Println(s, yhdbt.CheckPass(s))

	yhdbt.GLogin.Start()
	yhdbt.GKicked.Start()

	yhdbt.GRegistServer.Start()
}
