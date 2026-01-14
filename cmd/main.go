package main

import (
	"fmt"
	combinator "jabberwocky238/combinator/core"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 创建 gateway
	gateway := combinator.NewGateway("localhost:8899")

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
