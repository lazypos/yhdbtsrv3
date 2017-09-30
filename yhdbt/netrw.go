package yhdbt

import (
	"fmt"
	"net"
	"unsafe"
)

var (
	header_len = 12
	comm_flag  = 0x12348765
)

//基本发送函数
func sendMesssage(conn net.Conn, msg []byte) error {
	sendlen := 0
	for sendlen < len(msg) {
		n, err := conn.Write(msg[sendlen:])
		if err != nil {
			return fmt.Errorf(`[NET] sendMesssage error: %v`, err)
		} else if n <= 0 {
			return fmt.Errorf(`[NET] sendMesssage error: remote closed`)
		}
		sendlen += n
	}
	return nil
}

//基本接收函数
func recvMessage(conn net.Conn, totallen int) ([]byte, error) {
	recvlen := 0
	buf := make([]byte, totallen)
	for recvlen < totallen {
		n, err := conn.Read(buf[recvlen:])
		if err != nil {
			return []byte{}, fmt.Errorf(`[NET] recvMessage error: %v`, err)
		} else if n <= 0 {
			return []byte{}, fmt.Errorf(`[NET] recvMessage error: remote closed`)
		}
		recvlen += n
	}
	return buf, nil
}

//接受一个命令
func RecvCommond(conn net.Conn) (int, []byte, error) {
	head, err := recvMessage(conn, header_len)
	if err != nil {
		return 0, []byte{}, fmt.Errorf(`[NET] recv header error: %v`, err)
	}
	flags := int(*(*int32)(unsafe.Pointer(&head[0])))
	cmd := int(*(*int32)(unsafe.Pointer(&head[4])))
	datalen := int(*(*int32)(unsafe.Pointer(&head[8])))
	if flags != comm_flag || datalen < 0 || datalen > 1000 {
		return 0, []byte{}, fmt.Errorf(`[NET] recv header error: falg or length error.`)
	}
	content, err := recvMessage(conn, datalen)
	if err != nil {
		return 0, []byte{}, fmt.Errorf(`[NET] recv content error: %v`, err)
	}
	return cmd, content, nil
}

//发送一个命令
func SendCommond(conn net.Conn, content []byte) error {
	datalen := len(content)
	sendBuf := make([]byte, 8+datalen)

	*(*uint32)(unsafe.Pointer(&sendBuf[0])) = *(*uint32)(unsafe.Pointer(&comm_flag))
	*(*int32)(unsafe.Pointer(&sendBuf[4])) = *(*int32)(unsafe.Pointer(&datalen))

	copy(sendBuf[8:], content[:])
	if err := sendMesssage(conn, sendBuf); err != nil {
		return fmt.Errorf(`[NET] send content error %v`, err)
	}
	return nil
}
