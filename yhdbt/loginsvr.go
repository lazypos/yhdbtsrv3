package yhdbt

import (
	"fmt"
	"log"
	"net"
	"time"
)

type LoginServer struct {
}

func (this *LoginServer) Start(port string) error {
	listenfd, err := net.Listen("tcp", fmt.Sprintf(`:%s`, port))
	if err != nil {
		return fmt.Errorf(`[SEVER] listen failed: %v`, err)
	}
	go this.Routine_Listen(listenfd)
	return nil
}

func (this *LoginServer) Routine_Listen(serverfd net.Listener) {
	defer serverfd.Close()
	for {
		if conn, err := serverfd.Accept(); err != nil {
			log.Println(`[SERVER] accept error`, err)
			time.Sleep(time.Second)
		} else {
			log.Println(`[SERVER] recv connect`, conn.RemoteAddr().String())
			go this.processConn(conn)
		}
	}
}

func (this *LoginServer) processConn(conn net.Conn) {

	content, err := RecvCommond(conn)
	if err != nil {
		log.Println(`[SERVER] recv error:`, err)
		conn.Close()
		return
	}

	uid := GLogin.OnConnect(string(content[:]))
	if len(uid) == 0 {
		log.Println(`[SERVER] login error: can not find the loginkey.`)
		conn.Close()
		return
	}

}
