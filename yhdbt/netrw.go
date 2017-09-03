package yhdbt

import (
	"fmt"
	"net"
	"unsafe"
)

const (
	comm_flag = 0x12348765
)

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

func RecvCommon(conn net.Conn) ([]byte, error) {
	head, err := recvMessage(conn, 8)
	if err != nil {
		return []byte{}, fmt.Errorf(`[NET] recv header error:`, err)
	}
	flags := int(*(*int32)(unsafe.Pointer(&head[0])))
	datalen := int(*(*int32)(unsafe.Pointer(&head[4])))
	if flags != comm_flag || datalen < 0 || datalen > 1000 {
		return []byte{}, fmt.Errorf(`[NET] recv header error: falg or length error.`)
	}
	content, err := recvMessage(conn, datalen)
	if err != nil {
		return []byte{}, fmt.Errorf(`[NET] recv content error:`, err)
	}
	return content, nil
}
