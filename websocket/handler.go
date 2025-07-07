package websocket

import (
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
		conn.Close()
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
		s.conn.Close()
		s.terminal.Close()
	}
}

func (s *TerminalSession) Close() {
	s.close()
}

func (s *TerminalSession) handleInput() {
	defer s.wg.Done()
	defer s.Close()

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
			log.Println("WebSocket read:", string(data))
			if _, err := s.terminal.Write(data); err != nil {
				log.Println("Terminal write error:", err)
				return
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
			//output = bytes.Replace(output, []byte("\r\n"), []byte("\r"), -1)
			//log.Println("WebSocket output:", hex.Dump(output))
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
