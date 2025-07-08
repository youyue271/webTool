package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"webtool/terminal"
)

type ToolConfig struct {
	Name        string `yaml:"name"`
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
		msg := fmt.Sprintf("- %s: %s (使用: %s %s)\r\n", tool.Name, tool.Description, toolKey, tool.DefaultArgs)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}

	conn.WriteMessage(websocket.TextMessage, []byte("\r\n> "))

	for {
		_, input, err := conn.ReadMessage()
		if err != nil {
			break
		}

		inputStr := strings.TrimSpace(string(input))
		parts := strings.SplitN(inputStr, " ", 2)
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

		terminalId := fmt.Sprintf("tool-%d", time.Now().UnixNano())

		for client := range adminClients {
			creatCmd := fmt.Sprintf("CREATE_TOOL:%s|%s", tool.Name, fullCommand)
			client.WriteMessage(websocket.TextMessage, []byte(creatCmd))
		}

		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(
			"已创建工具终端: %s (%s)\r\n> ", tool.Name, terminalId)))
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

	session.wg.Add(3)
	go session.handleInput()
	go session.handleOutput()
	go session.execCommand(cmdName, cmdArgs)

	// 关闭监控
	go func() {
		session.wg.Wait()
		session.close()
	}()
}
func (s *TerminalSession) execCommand(cmdName string, cmdArgs []string) {
	s.wg.Done()
	cmd := cmdName + " " + strings.Join(cmdArgs, " ")
	_, err := s.terminal.Write([]byte(cmd))
	if err != nil {
		log.Println("Terminal write error:", err)
		return
	}
}
