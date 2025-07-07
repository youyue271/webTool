package main

import (
	"flag"
	"log"
	"net/http"

	"webtool/websocket"
)

func main() {
	port := flag.String("p", "8080", "port number")
	flag.Parse()

	// 添加健康检查端点
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/ws", websocket.Handler)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Printf("Server running on http://localhost:%s\n", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
