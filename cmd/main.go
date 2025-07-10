package main

import (
	"flag"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"

	"webtool/router"
	"webtool/websocket"
)

func main() {
	loadConfig("config.yaml")

	router.InitRouter()

	port := flag.String("p", "8080", "port number")
	flag.Parse()
	// 启动服务器
	log.Printf("Server running on http://localhost:%s\n", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
	// 添加健康检查端点

}

func loadConfig(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("配置文件读取失败: %v", err)
	}
	if err := yaml.Unmarshal(data, &websocket.Config); err != nil {
		log.Fatalf("配置文件解析失败: %v", err)
	}

}
