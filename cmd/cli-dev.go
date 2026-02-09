//go:build dev
// +build dev

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

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Failed to get user home directory: %v\n", err)
		return
	}
	combDir := filepath.Join(home, ".combinator")
	if err := os.MkdirAll(combDir, 0755); err != nil {
		fmt.Printf("Failed to create combinator directory: %v\n", err)
		return
	}

	var config common.DevConfig
	var realConfig common.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		fmt.Printf("Failed to parse config file: %v\n", err)
		return
	}

	fmt.Println("🔧 Development mode")

	// 转换所有 RDB 为本地 SQLite 文件
	for i := range config.Rdb {
		sqlitePath, err := getHomeDirWithSuffix("rdb", config.Rdb[i]+".sqlite")
		if err != nil {
			fmt.Printf("Failed to get SQLite path for RDB[%s]: %v\n", config.Rdb[i], err)
			return
		}
		realConfig.Rdb = append(realConfig.Rdb, common.RDBConfig{
			ID:  config.Rdb[i],
			URL: "sqlite://" + sqlitePath,
		})
		fmt.Printf("  ✓ RDB[%s] -> %s\n", config.Rdb[i], sqlitePath)
	}

	// 转换所有 KV 为内存模式
	for i := range config.Kv {
		realConfig.Kv = append(realConfig.Kv, common.KVConfig{
			ID:  config.Kv[i],
			URL: "memory://",
		})
		fmt.Printf("  ✓ KV[%s] -> %s\n", config.Kv[i], "memory://")
	}

	// 转换所有 S3 为本地目录
	for i := range config.S3 {
		s3Path, err := getHomeDirWithSuffix("s3", config.S3[i])
		if err != nil {
			fmt.Printf("Failed to get path for S3[%s]: %v\n", config.S3[i], err)
			return
		}
		if err := os.MkdirAll(s3Path, 0755); err != nil {
			fmt.Printf("Failed to create directory for S3[%s]: %v\n", config.S3[i], err)
			return
		}
		localS3 := common.S3Config{
			ID:  config.S3[i],
			URL: "local://" + s3Path,
		}
		realConfig.S3 = append(realConfig.S3, localS3)
		fmt.Printf("  ✓ S3[%s] -> %s\n", config.S3[i], s3Path)
	}

	// 启动网关
	gateway := combinator.NewGateway(&realConfig, true)
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

func getHomeDirWithSuffix(suffixs ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取 HOME 目录: %w", err)
	}
	return filepath.Join(home, ".combinator", filepath.Join(suffixs...)), nil
}

func runDevListRdb(cmd *cobra.Command, args []string) {
	rdbDir, err := getHomeDirWithSuffix("rdb")
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
	rdbDir, err := getHomeDirWithSuffix("rdb")
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
