package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	combinator "jabberwocky238/combinator/core"
	common "jabberwocky238/combinator/core/common"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var (
	devConfigPath string
	devListenAddr string
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "启动 Combinator 开发模式（自动转换为内存数据库）",
	Run:   runDev,
}

func init() {
	devCmd.Flags().StringVarP(&devConfigPath, "config", "c", "config.combinator.json", "配置文件路径")
	devCmd.Flags().StringVarP(&devListenAddr, "listen", "l", "localhost:8899", "监听地址")
}

func runDev(cmd *cobra.Command, args []string) {
	// 加载配置文件
	configJSON, err := os.ReadFile(devConfigPath)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		return
	}

	var config common.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return
	}

	// 转换所有非 SQLite 数据库为内存 SQLite
	fmt.Println("🔧 Development mode: Converting databases to in-memory SQLite...")
	for i := range config.Rdb {
		url := config.Rdb[i].URL
		if !strings.HasPrefix(url, "sqlite://") {
			oldURL := url
			config.Rdb[i].URL = "sqlite://:memory:"
			fmt.Printf("  ✓ RDB[%s]: %s -> sqlite://:memory:\n", config.Rdb[i].ID, oldURL)
		} else {
			fmt.Printf("  - RDB[%s]: %s (unchanged)\n", config.Rdb[i].ID, url)
		}
	}

	// 启动网关
	gateway := combinator.NewGateway(&config, true)
	gateway.SetupMonitorAPI()

	// 启动信号监听
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中启动 gateway
	go func() {
		fmt.Printf("🚀 Starting development server on %s...\n", devListenAddr)
		if err := gateway.Start(devListenAddr); err != nil {
			fmt.Printf("Gateway error: %v\n", err)
			os.Exit(1)
		}
	}()

	// 阻塞等待 Ctrl+C
	<-sigChan
	fmt.Println("\n✓ Received interrupt signal, shutting down gracefully...")
}

func cors(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	})
}
