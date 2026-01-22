package main

import (
	"encoding/json"
	"fmt"
	combinator "jabberwocky238/combinator/core"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 加载配置文件（如果不存在则使用默认配置）
	configPath := "config.example.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	configJSON, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config combinator.Config
	err = json.Unmarshal(configJSON, &config)

	gateway := combinator.NewGateway(&config)

	// 启动信号监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动 gateway
	go func() {
		fmt.Println("Starting gateway server...")
		if err := gateway.Start("localhost:8899"); err != nil {
			fmt.Printf("Gateway error: %v\n", err)
			os.Exit(1)
		}
	}()

	// 阻塞等待 Ctrl+C
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
}
