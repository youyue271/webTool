package websocket

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

	"webtool/terminal"
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

func AdminWebSocketHandler(w http.ResponseWriter, r *http.Request) {
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
			fullCommand := strings.Replace(tool.Command, "{{args}}", args, -1)
			fullCommand = strings.Replace(fullCommand, "{{exePath}}", tool.ExePath, -1)
			terminalId := fmt.Sprintf("tool-%d", time.Now().UnixNano())
			for client := range adminClients {
				creatCmd := fmt.Sprintf("CREATE_TOOL::%s|%s", tool.Name, fullCommand)
				//log.Println(creatCmd)
				client.WriteMessage(websocket.TextMessage, []byte(creatCmd))
			}
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(
				"已创建工具终端: %s (%s)\r\n> ", tool.Name, terminalId)))
		} else {
			inputStr = append(inputStr, input...)
		}
	}

}
func ToolWebSocketHandler(w http.ResponseWriter, r *http.Request) {
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

	terminalId := r.URL.Query().Get("terminalId")
	command := r.URL.Query().Get("command")

	//log.Println("terminalId:", terminalId, "command:", command)
	if terminalId == "" || command == "" {
		conn.Close()
		return
	}

	clientsMutex.Lock()
	toolClients[terminalId] = conn
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(toolClients, terminalId)
		clientsMutex.Unlock()
		conn.Close()
	}()

	go runToolCommand(conn, command)

	// 可以用于处理可鞥的输入输出 TODO: 处理各个文件的输出内容
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func runToolCommand(conn *websocket.Conn, command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		conn.WriteMessage(websocket.TextMessage, []byte("错误: 空指令\r\n"))
		return
	}

	cmdName := parts[0]
	cmdArgs := parts[1:]

	term, err := terminal.NewSystemTerminal()
	if err != nil {
		return
	}

	session := &TerminalSession{
		conn:     conn,
		terminal: term,
		closed:   make(chan struct{}),
	}

	session.wg.Add(2)
	go session.handleOutput()
	go session.execCommand(cmdName, cmdArgs)
	go session.handleInput()
	//

	// 关闭监控
	go func() {
		session.wg.Wait()
		session.close()
	}()
}
