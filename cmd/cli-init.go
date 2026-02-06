package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initForce bool
)

var defaultConfig = map[string]any{
	"rdb": []string{"rdb2077"},
	"kv":  []string{"kv114514"},
	// "s3":  []string{"s31919810"},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new combinator configuration file",
	Run:   runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing configuration file")
}

func runInit(cmd *cobra.Command, args []string) {
	configPath := "config.combinator.json"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		fmt.Printf("Error: %s already exists. Use --force to overwrite.\n", configPath)
		os.Exit(1)
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		fmt.Printf("Error creating configuration: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	err = os.WriteFile(configPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing configuration file: %v\n", err)
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(configPath)
	fmt.Printf("✓ Configuration file created: %s\n", absPath)
}
