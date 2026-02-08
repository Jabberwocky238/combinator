package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	combinator "jabberwocky238/combinator/core"
	common "jabberwocky238/combinator/core/common"

	"github.com/spf13/cobra"
)

var (
	devConfigPath string
	devListenAddr string
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "开发模式相关命令",
	Run:   runDev,
}

var devClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "清理开发缓存",
}

var devClearRdbCmd = &cobra.Command{
	Use:   "rdb [id]",
	Short: "清理 RDB 缓存文件",
	Args:  cobra.MaximumNArgs(1),
	Run:   runDevClearRdb,
}

var devListCmd = &cobra.Command{
	Use:   "list",
	Short: "查看开发缓存",
}

var devListRdbCmd = &cobra.Command{
	Use:   "rdb",
	Short: "查看 RDB 缓存文件",
	Run:   runDevListRdb,
}

func init() {
	devCmd.Flags().StringVarP(&devConfigPath, "config", "c", "config.combinator.json", "配置文件路径")
	devCmd.Flags().StringVarP(&devListenAddr, "listen", "l", "localhost:8899", "监听地址")

	devClearCmd.AddCommand(devClearRdbCmd)
	devListCmd.AddCommand(devListRdbCmd)
	devCmd.AddCommand(devClearCmd)
	devCmd.AddCommand(devListCmd)

	// 自动注册到 root
	rootCmd.AddCommand(devCmd)
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

	// 获取 rdb 存储目录
	home, err2 := os.UserHomeDir()
	if err2 != nil {
		fmt.Printf("Failed to get home directory: %v\n", err2)
		return
	}
	rdbDir := filepath.Join(home, ".combinator", "rdb")
	if err := os.MkdirAll(rdbDir, 0755); err != nil {
		fmt.Printf("Failed to create rdb directory: %v\n", err)
		return
	}

	// 转换所有 RDB 为本地 SQLite 文件
	fmt.Println("🔧 Development mode")

	for i := range config.Rdb {
		oldURL := config.Rdb[i].URL
		sqlitePath := filepath.Join(rdbDir, config.Rdb[i].ID+".sqlite")
		config.Rdb[i].URL = "sqlite://" + sqlitePath
		fmt.Printf("  ✓ RDB[%s]: %s -> sqlite://%s\n", config.Rdb[i].ID, oldURL, sqlitePath)
	}

	// 转换所有 KV 为内存模式
	for i := range config.Kv {
		oldURL := config.Kv[i].URL
		config.Kv[i].URL = "memory://"
		fmt.Printf("  ✓ KV[%s]: %s -> memory://\n", config.Kv[i].ID, oldURL)
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

func getRdbDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取 HOME 目录: %w", err)
	}
	return filepath.Join(home, ".combinator", "rdb"), nil
}

func runDevListRdb(cmd *cobra.Command, args []string) {
	rdbDir, err := getRdbDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	entries, err := os.ReadDir(rdbDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("没有 RDB 缓存文件")
			return
		}
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sqlite") {
			continue
		}
		info, _ := entry.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		id := strings.TrimSuffix(entry.Name(), ".sqlite")
		fmt.Printf("  [%s] %s  (%d bytes)\n", id, filepath.Join(rdbDir, entry.Name()), size)
		count++
	}

	if count == 0 {
		fmt.Println("没有 RDB 缓存文件")
	} else {
		fmt.Printf("\n共 %d 个 RDB 缓存文件\n", count)
	}
}

func runDevClearRdb(cmd *cobra.Command, args []string) {
	rdbDir, err := getRdbDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(args) == 1 {
		// 删除指定 id
		id := args[0]
		target := filepath.Join(rdbDir, id+".sqlite")
		if _, err := os.Stat(target); os.IsNotExist(err) {
			fmt.Printf("RDB 缓存不存在: %s\n", target)
			return
		}
		fmt.Printf("确认删除 RDB[%s] (%s)? (y/yes): ", id, target)
		var confirm string
		fmt.Scanln(&confirm)
		confirm = strings.ToLower(strings.TrimSpace(confirm))
		if confirm != "y" && confirm != "yes" {
			fmt.Println("已取消")
			return
		}
		if err := os.Remove(target); err != nil {
			fmt.Printf("删除失败: %v\n", err)
			return
		}
		fmt.Printf("✓ 已删除 RDB[%s]\n", id)
		return
	}

	// 删除全部
	entries, err := os.ReadDir(rdbDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("没有 RDB 缓存文件")
			return
		}
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sqlite") {
			files = append(files, entry.Name())
		}
	}
	if len(files) == 0 {
		fmt.Println("没有 RDB 缓存文件")
		return
	}

	fmt.Printf("将删除以下 %d 个 RDB 缓存:\n", len(files))
	for _, f := range files {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Print("确认删除? (y/yes): ")
	var confirm string
	fmt.Scanln(&confirm)
	confirm = strings.ToLower(strings.TrimSpace(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("已取消")
		return
	}

	for _, f := range files {
		if err := os.Remove(filepath.Join(rdbDir, f)); err != nil {
			fmt.Printf("删除 %s 失败: %v\n", f, err)
		} else {
			fmt.Printf("✓ %s\n", f)
		}
	}
	fmt.Println("清理完成")
}
