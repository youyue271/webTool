package controller

import (
	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"webtool/internal/service"
	"webtool/internal/terminal"
)

type ToolConfig struct {
	Name        string `yaml:"name"`
	ExePath     string `yaml:"exePath"`
	Command     string `yaml:"command"`
	Description string `yaml:"description"`
	DefaultArgs string `yaml:"defaultArgs"`
}

type AppConfig struct {
	Tools map[string]ToolConfig `yaml:"tools"`
}

var (
	adminClients = make(map[*websocket.Conn]bool)
	toolClients  = make(map[string]*websocket.Conn)
	clientsMutex sync.Mutex
	Config       AppConfig
)

func AdminTerminalHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("管理WebSocket升级失败", err)
		return
	}

	clientsMutex.Lock()
	adminClients[conn] = true
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(adminClients, conn)
		clientsMutex.Unlock()
		err := conn.Close()
		if err != nil {
			return
		}
	}()

	conn.WriteMessage(websocket.TextMessage, []byte("管理控制台已链接\n\r"))
	conn.WriteMessage(websocket.TextMessage, []byte("可用工具: \n\r"))

	for toolKey, tool := range Config.Tools {
		_, err := os.Lstat(tool.ExePath)
		if !os.IsNotExist(err) {
			msg := fmt.Sprintf("- %s\r\n", toolKey)
			conn.WriteMessage(websocket.TextMessage, []byte(msg))
		}
	}

	conn.WriteMessage(websocket.TextMessage, []byte("\r\n> "))
	inputStr := make([]byte, 1024)
	inputStr = nil
	for {
		_, input, err := conn.ReadMessage()
		//log.Printf("接收输入: %v\n", hex.Dump(input[len(input)-1:]))
		if err != nil {
			break
		}
		if bytes.Equal(input, []byte("\x7f")) {
			if len(inputStr) == 0 {
				continue
			}
			inputStr = inputStr[:len(inputStr)-1]
			//log.Println("buff read:", string(inputStr))
		} else if bytes.Equal(input[len(input)-1:], []byte("\n")) || bytes.Equal(input[len(input)-1:], []byte("\r")) {
			parts := strings.SplitN(string(inputStr), " ", 2)
			//log.Println(hex.Dump(inputStr))
			inputStr = nil
			toolKey := parts[0]
			args := ""
			if len(parts) > 1 {
				args = parts[1]
			}
			tool, exists := Config.Tools[toolKey]
			if !exists {
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("未知工具")))
				continue
			}
			if args == "" && tool.DefaultArgs != "" {
				args = tool.DefaultArgs
			}
			name := tool.Name
			fullCommand := strings.Replace(tool.Command, "{{args}}", args, -1)
			fullCommand = strings.Replace(tool.Command, "{{name}}", name, -1)
			fullCommand = strings.Replace(fullCommand, "{{exePath}}", tool.ExePath, -1)
			terminalId := fmt.Sprintf("tool-%d", time.Now().UnixNano())
			for client := range adminClients {
				creatCmd := fmt.Sprintf("CREATE_TOOL::%s|%s|%s", tool.ExePath, toolKey, fullCommand)
				//log.Println(creatCmd)
				err := client.WriteMessage(websocket.TextMessage, []byte(creatCmd))
				if err != nil {
					return
				}
			}
			err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(
				"已创建工具终端: %s (%s)\r\n> ", toolKey, terminalId)))
			if err != nil {
				return
			}
		} else {
			inputStr = append(inputStr, input...)
		}
	}

}

func ToolTerminalHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("工具WebSocket升级失败:", err)
		return
	}
	//println(1)
	terminalId := r.URL.Query().Get("terminalId")
	command := r.URL.Query().Get("command")
	exePath := r.URL.Query().Get("exePath")

	print(terminalId, command)
	//log.Println("terminalId:", terminalId, "command:", command)
	if terminalId == "" || command == "" {
		conn.Close()
		return
	}

	clientsMutex.Lock()
	toolClients[terminalId] = conn
	clientsMutex.Unlock()

	//var wg sync.WaitGroup
	//println(2)
	runToolCommand(conn, command, exePath)
	defer func() {
		//println(4)
		clientsMutex.Lock()
		delete(toolClients, terminalId)
		clientsMutex.Unlock()
		conn.Close()
	}()
	//println(3)
	log.Printf("Tool Terminal session %v closed", terminalId)
}

func runToolCommand(conn *websocket.Conn, command string, exePath string) {
	//defer wg.Done()
	term, err := terminal.NewSystemTerminal()
	if err != nil {
		return
	}

	session := service.CreateTerminalSession(conn, term)

	cdCommand := "cd \"" + exePath + "\"\n"
	session.ExecCommand(cdCommand)
	command = command + "\n"
	session.ExecCommand(command)

	session.Listen()
}

func NewTerminalSession(w http.ResponseWriter, r *http.Request) (*service.TerminalSession, error) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

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

	session := service.CreateTerminalSession(conn, term)

	session.Listen()
	return session, nil
}

func ExecTerminalHandler(w http.ResponseWriter, r *http.Request) {
	session, err := NewTerminalSession(w, r)
	if err != nil {
		log.Println("Failed to create terminal session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer session.Close()
	log.Println("Terminal session closed")
}
