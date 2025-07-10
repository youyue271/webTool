package router

import (
	"net/http"
	"webtool/websocket"
)

func InitRouter() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.Handle("/node_modules", http.FileServer(http.Dir("./node_modules")))

	http.HandleFunc("/ws", websocket.Handler)
	http.HandleFunc("/ws-admin", websocket.AdminWebSocketHandler)
	http.HandleFunc("/ws-tool", websocket.ToolWebSocketHandler)
}
