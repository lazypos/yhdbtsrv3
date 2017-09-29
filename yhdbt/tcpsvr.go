package yhdbt

import (
	"fmt"
	"log"
	"net"
	"time"
)

// TCP服务器
type TCPServer struct {
}

var GTCPServer = &TCPServer{}

func (this *TCPServer) Start(port string) error {
	listenfd, err := net.Listen("tcp", fmt.Sprintf(`:%s`, port))
	if err != nil {
		return fmt.Errorf(`[SEVER] listen failed: %v`, err)
	}
	go this.Routine_Listen(listenfd)
	return nil
}

//监听线程
func (this *TCPServer) Routine_Listen(serverfd net.Listener) {
	defer serverfd.Close()
	for {

		if conn, err := serverfd.Accept(); err != nil {
			log.Println(`[SERVER] accept error`, err)
			time.Sleep(time.Second)
		} else {
			log.Println(`[SERVER] recv connect`, conn.RemoteAddr().String())
			conn.(*net.TCPConn).SetLinger(0)
			go this.processConn(conn)
		}
	}
}

//处理连接，将合法玩家输送给游戏大厅
func (this *TCPServer) processConn(conn net.Conn) {

	cmd, content, err := RecvCommond(conn)
	if err != nil {
		log.Println(`[SERVER] recv error:`, err)
		conn.Close()
		return
	}
	if cmd != cmd_add_hall {
		log.Println(`[SERVER] recv CMD unknow.`)
		conn.Close()
		return
	}
	// 先通过login key 得到uid
	uid := GLogin.OnConnect(string(content[:]))
	if len(uid) == 0 {
		log.Println(`[SERVER] login error: can not find the loginkey.`)
		conn.Close()
		return
	}
	// 根据UID 得到玩家信息
	pInfo := GLogin.GetPlayerInfo(conn, uid)
	if pInfo == nil {
		log.Println(`[SERVER] login error: can not find the player info from uid.`)
		return
	}
	pInfo.SendMessage(fmt.Sprintf(fmt_plyer_info, pInfo.NickName, pInfo.Score, pInfo.Win, pInfo.Lose, pInfo.Run))
	// 加入游戏大厅
	GHall.AddPlayer(pInfo)
}
