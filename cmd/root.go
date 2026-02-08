package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "combinator",
	Short: "Combinator - 多数据库网关服务",
	Long:  "Combinator 是一个统一的 HTTP API 网关，支持多种数据库后端（RDB 和 KV）",
}

func Execute() error {
	return rootCmd.Execute()
}
