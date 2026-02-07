package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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
	startCmd.Flags().StringVarP(&watchMode, "watch", "w", "", "配置监听模式: file, api, all")
	startCmd.Flags().IntVar(&watchInterval, "watch-interval", 5, "文件监听间隔（秒）")
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

// 文件监听
func (s *StartCmd) watchConfigFile(path string, interval int, reloadChan chan<- *common.Config) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 直接读文件内容以避免某些文件系统不更新修改时间的问题
		config, newHash, err := s.loadConfig(path)
		if err != nil {
			fmt.Printf("⚠️  Failed to read config file: %v\n", err)
			continue
		}

		// 使用读写锁安全地读取lastHash
		s.lastHashMu.RLock()
		currentHash := s.lastHash
		s.lastHashMu.RUnlock()

		if newHash == currentHash {
			continue // 文件内容未变更
		}

		fmt.Println("📝 Config file changed, reloading...")

		// 更新hash（在发送到channel之前）
		s.lastHashMu.Lock()
		s.lastHash = newHash
		s.lastHashMu.Unlock()

		reloadChan <- config
	}
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

	// 配置重载通道
	reloadChan := make(chan *common.Config, 1)

	// 启动 watch 模式
	if watchMode == "file" || watchMode == "all" {
		fmt.Printf("📁 File watch enabled (interval: %ds)\n", watchInterval)
		go s.watchConfigFile(configPath, watchInterval, reloadChan)
	}

	if watchMode == "api" || watchMode == "all" {
		fmt.Println("🌐 API reload endpoint enabled at /reload")
		gateway.SetupReloadAPI(reloadChan)
	}

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
	for {
		select {
		case <-sigChan:
			fmt.Println("\n✓ Received interrupt signal, shutting down gracefully...")
			return
		case newConfig := <-reloadChan:
			fmt.Println("✅ Reloading gateway with new configuration...")
			if err := gateway.Reload(newConfig); err != nil {
				fmt.Printf("❌ Failed to reload gateway: %v\n", err)
			} else {
				fmt.Println("✅ Gateway reloaded successfully")
			}
		}
	}
}
