package main

import (
	"encoding/json"
	"fmt"
	"time"

	combinator "jabberwocky238/combinator/core"
	common "jabberwocky238/combinator/core/common"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	configPath    string
	listenAddr    string
	watchMode     string
	watchInterval int
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Combinator 网关服务",
	Run:   runStart,
}

func init() {
	startCmd.Flags().StringVarP(&configPath, "config", "c", "config.combinator.json", "配置文件路径")
	startCmd.Flags().StringVarP(&listenAddr, "listen", "l", "localhost:8899", "监听地址")
	startCmd.Flags().StringVarP(&watchMode, "watch", "w", "", "配置监听模式: file, api, all")
	startCmd.Flags().IntVar(&watchInterval, "watch-interval", 5, "文件监听间隔（秒）")
}

// 加载配置文件
func loadConfig(path string) (*common.Config, error) {
	configJSON, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config common.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// 文件监听
func watchConfigFile(path string, interval int, reloadChan chan<- *common.Config) {
	var lastModTime time.Time

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("⚠️  Failed to stat config file: %v\n", err)
			continue
		}

		if info.ModTime().After(lastModTime) {
			lastModTime = info.ModTime()
			fmt.Println("📝 Config file changed, reloading...")

			config, err := loadConfig(path)
			if err != nil {
				fmt.Printf("❌ Failed to reload config: %v\n", err)
				continue
			}

			reloadChan <- config
		}
	}
}

func runStart(cmd *cobra.Command, args []string) {
	// 加载初始配置
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	gateway := combinator.NewGateway(config)

	// 配置重载通道
	reloadChan := make(chan *common.Config, 1)

	// 启动 watch 模式
	if watchMode == "file" || watchMode == "all" {
		fmt.Printf("📁 File watch enabled (interval: %ds)\n", watchInterval)
		go watchConfigFile(configPath, watchInterval, reloadChan)
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
			gateway.Reload(newConfig)
		}
	}
}
