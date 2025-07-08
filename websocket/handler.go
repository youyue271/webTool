package websocket

import (
	"bytes"
	"encoding/hex"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"webtool/terminal"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type TerminalSession struct {
	conn     *websocket.Conn
	terminal *terminal.Terminal
	wg       sync.WaitGroup
	closed   chan struct{}
}

func NewTerminalSession(w http.ResponseWriter, r *http.Request) (*TerminalSession, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	term, err := terminal.NewSystemTerminal()
	if err != nil {
		err := conn.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	session := &TerminalSession{
		conn:     conn,
		terminal: term,
		closed:   make(chan struct{}),
	}

	session.wg.Add(2)
	go session.handleInput()
	go session.handleOutput()

	// 关闭监控
	go func() {
		session.wg.Wait()
		session.close()
	}()

	return session, nil
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

func (s *TerminalSession) handleInput() {
	defer s.wg.Done()
	defer s.Close()
	buff := make([]byte, 1024)
	buff = nil
	var buffSize = 0
	for {
		select {
		case <-s.closed:
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
			return
		case output, ok := <-s.terminal.Read():
			if !ok {
				return
			}

			//log.Println("WebSocket output:", hex.Dump(output))
			output = bytes.Replace(output, []byte("\r\n"), []byte("\r"), -1)
			//log.Println("WebSocket output:", hex.Dump(output))
			//output = append(output, []byte("PS> ")...)
			if err := s.conn.WriteMessage(websocket.TextMessage, output); err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}
	}
}

func Handler(w http.ResponseWriter, r *http.Request) {
	session, err := NewTerminalSession(w, r)
	if err != nil {
		log.Println("Failed to create terminal session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer session.Close()

	<-session.closed
	log.Println("Terminal session closed")
}
