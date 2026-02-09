//go:build prod
// +build prod

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	combinator "jabberwocky238/combinator/core"
	common "jabberwocky238/combinator/core/common"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

type StartCmd struct {
	lastHashMu sync.RWMutex
	lastHash   [32]byte
}

var (
	configPath       string
	listenAddr       string
	watchMode        string
	watchInterval    int
	startCmdInstance StartCmd
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Combinator 网关服务",
	Run:   startCmdInstance.runStart,
}

func init() {
	startCmd.Flags().StringVarP(&configPath, "config", "c", "config.combinator.json", "配置文件路径")
	startCmd.Flags().StringVarP(&listenAddr, "listen", "l", "localhost:8899", "监听地址")

	// 自动注册到 root (仅在 prod 模式下)
	rootCmd.AddCommand(startCmd)
}

// 加载配置文件
func (s *StartCmd) loadConfig(path string) (*common.Config, [32]byte, error) {
	configJSON, err := os.ReadFile(path)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("failed to read config file: %w", err)
	}

	newHash := sha256.Sum256(configJSON)
	var config common.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, newHash, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, newHash, nil
}

func (s *StartCmd) runStart(cmd *cobra.Command, args []string) {
	// 加载初始配置
	config, newHash, err := s.loadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}
	s.lastHash = newHash

	// 创建并启动 gateway
	gateway := combinator.NewGateway(config, false)

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

	// 主循环：监听信号和配置重载
	<-sigChan
	fmt.Println("\n✓ Received interrupt signal, shutting down gracefully...")
}
