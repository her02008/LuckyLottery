package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"lottery-tool/internal/config"
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "lottery-tool",
		Short: "大乐透和双色球选号工具",
		Long:  `一个基于历史数据和AI分析的大乐透和双色球选号工具`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/config.yaml", "配置文件路径")
}

func initConfig() {
	if _, err := config.Load(configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
