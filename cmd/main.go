package main

import (
	"encoding/json"
	"flag"
	"fmt"
	combinator "jabberwocky238/combinator/core"
	common "jabberwocky238/combinator/core/common"
	"os"
	"os/signal"
	"syscall"
)

var configPath = flag.String("c", "config.json", "配置文件路径")
var listenAddr = flag.String("l", "localhost:8899", "监听地址")

func cmdParsing() {
	// 解析命令行参数
	flag.StringVar(configPath, "config", "config.json", "配置文件路径")
	flag.StringVar(listenAddr, "listen", "localhost:8899", "监听地址")
	flag.Parse()
}

func main() {
	cmdParsing()
	// 加载配置文件
	configJSON, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config common.Config
	err = json.Unmarshal(configJSON, &config)

	gateway := combinator.NewGateway(&config)

	// 启动信号监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动 gateway
	go func() {
		fmt.Printf("Starting gateway server on %s...\n", *listenAddr)
		if err := gateway.Start(*listenAddr); err != nil {
			fmt.Printf("Gateway error: %v\n", err)
			os.Exit(1)
		}
	}()

	// 阻塞等待 Ctrl+C
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
}
