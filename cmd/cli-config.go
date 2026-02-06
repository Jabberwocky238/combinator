package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type GlobalConfig struct {
	UserUID string `json:"useruid"`
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("无法获取 HOME 目录: %w", err)
	}
	return filepath.Join(home, ".combinator"), nil
}

func getConfigFilePath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadGlobalConfig() (*GlobalConfig, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %w", err)
	}
	var config GlobalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("配置文件格式错误: %w", err)
	}
	return &config, nil
}

func saveGlobalConfig(config *GlobalConfig) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理全局配置 (~/.combinator/config.json)",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化全局配置文件",
	Run:   runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项",
	Args:  cobra.ExactArgs(2),
	Run:   runConfigSet,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	dir, err := getConfigDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	configPath, _ := getConfigFilePath()

	// 检查是否已存在
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("配置文件已存在: %s\n", configPath)
		fmt.Println("如需重新初始化，请先手动删除该文件")
		return
	}

	// 创建目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		os.Exit(1)
	}

	// 写入默认配置
	defaultConfig := &GlobalConfig{
		UserUID: "",
	}
	if err := saveGlobalConfig(defaultConfig); err != nil {
		fmt.Printf("写入配置文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 全局配置已初始化: %s\n", configPath)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	config, err := loadGlobalConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("请先运行 combinator config init")
		os.Exit(1)
	}

	switch key {
	case "useruid":
		config.UserUID = value
	default:
		fmt.Printf("未知的配置项: %s\n", key)
		fmt.Println("可用配置项: useruid")
		os.Exit(1)
	}

	if err := saveGlobalConfig(config); err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ %s = %s\n", key, value)
}
