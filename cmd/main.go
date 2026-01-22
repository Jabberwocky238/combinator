package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"jabberwocky238/combinator/core/config"
	"jabberwocky238/combinator/core/manager"
)

func main() {
	// 加载配置文件（如果不存在则使用默认配置）
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := loadOrCreateDefaultConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}


	// 启动信号监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动 gateway
	go func() {
		fmt.Println("Starting gateway server...")
		if err := gateway.Start(); err != nil {
			fmt.Printf("Gateway error: %v\n", err)
			os.Exit(1)
		}
	}()

	// 阻塞等待 Ctrl+C
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
}
