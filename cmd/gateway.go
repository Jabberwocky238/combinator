package main

import (
	"encoding/json"
	"fmt"
	// "io"
	// "net/http"
	// "time"

	combinator "jabberwocky238/combinator/core"
	common "jabberwocky238/combinator/core/common"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	configPath string
	listenAddr string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Combinator 网关服务",
	Run:   runStart,
}

func init() {
	startCmd.Flags().StringVarP(&configPath, "config", "c", "config.json", "配置文件路径")
	startCmd.Flags().StringVarP(&listenAddr, "listen", "l", "localhost:8899", "监听地址")
}

func runStart(cmd *cobra.Command, args []string) {
	// 加载配置文件
	configJSON, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config common.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return
	}

	gateway := combinator.NewGateway(&config)

	// 启动信号监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动 gateway
	go func() {
		fmt.Printf("Starting gateway server on %s...\n", listenAddr)
		if err := gateway.Start(listenAddr); err != nil {
			fmt.Printf("Gateway error: %v\n", err)
			os.Exit(1)
		}
	}()

	// 阻塞等待 Ctrl+C
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
}
