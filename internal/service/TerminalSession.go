package service

import (
	"bytes"
	"encoding/hex"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"webtool/internal/terminal"
)

type TerminalSession struct {
	conn     *websocket.Conn
	terminal *terminal.Terminal
	wg       sync.WaitGroup
	closed   chan struct{}
}

func CreateTerminalSession(conn *websocket.Conn, terminal *terminal.Terminal) *TerminalSession {
	return &TerminalSession{
		conn:     conn,
		terminal: terminal,
		closed:   make(chan struct{}),
	}
}

func (s *TerminalSession) close() {
	select {
	case <-s.closed:
		return
	default:
		close(s.closed)
		err := s.conn.Close()
		if err != nil {
			return
		}
		s.terminal.Close()
	}
}

func (s *TerminalSession) Close() {
	s.close()
}

func (s *TerminalSession) Listen() {
	s.wg.Add(2)
	go s.handleInput()
	go s.handleOutput()

	// 关闭监控
	defer func() {
		s.wg.Wait()
		s.close()
	}()

}

func (s *TerminalSession) handleInput() {
	defer s.wg.Done()
	defer s.Close()
	buff := make([]byte, 1024)
	buff = nil
	var buffSize = 0
	for {
		select {
		case <-s.closed:
			//println(9)
			return
		default:
			_, data, err := s.conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Println("WebSocket read error:", err)
				}
				return
			}
			log.Println("WebSocket read:", hex.Dump(data))
			if bytes.Equal(data, []byte("\x7f")) {
				if buffSize == 0 {
					continue
				}
				buff = buff[:buffSize-1]
				buffSize -= 1
				log.Println("buff read:", string(buff))
			} else if bytes.Equal(data, []byte("\n")) {
				buff = append(buff, data...)
				log.Println("buff read:", hex.Dump(buff))
				if _, err := s.terminal.Write(buff); err != nil {
					log.Println("Terminal write error:", err)
					return
				}
				buffSize = 0
				buff = nil
			} else {
				buff = append(buff, data...)
				buffSize += len(data)
				log.Println("buff read:", string(buff))
			}

		}
	}
}

func (s *TerminalSession) handleOutput() {
	defer s.wg.Done()
	defer s.Close()

	for {
		select {
		case <-s.closed:
			//println(8)
			return
		case output, ok := <-s.terminal.Read():
			if !ok {
				return
			}

			log.Println("WebSocket output:", hex.Dump(output))

			//output = bytes.Replace(output, []byte(" \r\n"), []byte(" \r"), -1)
			//output = bytes.Replace(output, []byte("\r\n"), []byte("\r"), -1)
			log.Println("WebSocket change:", hex.Dump(output))
			//output = append(output, []byte("PS> ")...)
			if bytes.Equal(output, []byte("\r\n")) {
				_ = s.conn.WriteMessage(websocket.TextMessage, []byte("\r     \r"))
				continue
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, output); err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}
	}
}

func (s *TerminalSession) ExecCommand(cmd string) {
	//cmd := cmdName + " " + strings.Join(cmdArgs, " ")
	//cmd = strings.Trim(cmd, " ") + "\n"
	log.Printf("cmd: %v\n", hex.Dump([]byte(cmd)))
	_, err := s.terminal.Write([]byte(cmd))
	if err != nil {
		log.Println("Terminal write error:", err)
		return
	}
}
