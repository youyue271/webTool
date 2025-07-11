package router

import (
	"net/http"
	"webtool/internal/controller"
)

func InitRouter() {
	http.HandleFunc("/health", controller.HealthCheck)

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.Handle("/node_modules", http.FileServer(http.Dir("./node_modules")))

	// 通信ws
	http.HandleFunc("/ws-exec-terminal", controller.ExecTerminalHandler)
	http.HandleFunc("/ws-admin-terminal", controller.AdminTerminalHandler)
	http.HandleFunc("/ws-tool-terminal", controller.ToolTerminalHandler)
}
