package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"lottery-tool/internal/config"
	"lottery-tool/internal/predictor"
	"lottery-tool/internal/scraper"
	"lottery-tool/internal/storage"
	"lottery-tool/pkg/types"
)

// fetchCmd 抓取数据命令
var fetchCmd = &cobra.Command{
	Use:   "fetch [dlt|ssq]",
	Short: "抓取历史开奖数据",
	Long:  `从网络抓取大乐透或双色球的历史开奖数据`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lotteryType := types.LotteryType(args[0])
		if lotteryType != types.LotteryTypeDLT && lotteryType != types.LotteryTypeSSQ {
			fmt.Println("错误：彩票类型必须是 dlt(大乐透) 或 ssq(双色球)")
			return
		}

		limit, _ := cmd.Flags().GetInt("limit")
		
		cfg := config.Get()
		
		// 创建爬虫
		factory := scraper.NewScraperFactory(cfg.Scraper.DLTURL, cfg.Scraper.SSQURL)
		s, err := factory.Create(lotteryType)
		if err != nil {
			fmt.Printf("创建爬虫失败: %v\n", err)
			return
		}

		fmt.Printf("正在抓取%s历史数据...\n", getLotteryName(lotteryType))
		
		results, err := s.FetchHistory(limit)
		if err != nil {
			fmt.Printf("抓取数据失败: %v\n", err)
			return
		}

		// 保存到数据库
		store, err := storage.New(cfg.Database.Path)
		if err != nil {
			fmt.Printf("连接数据库失败: %v\n", err)
			return
		}
		defer store.Close()

		savedCount := 0
		for _, result := range results {
			if err := store.SaveDrawResult(result); err != nil {
				fmt.Printf("保存数据失败: %v\n", err)
				continue
			}
			savedCount++
		}

		fmt.Printf("成功抓取并保存 %d 条数据\n", savedCount)
	},
}

// predictCmd 预测命令
var predictCmd = &cobra.Command{
	Use:   "predict [dlt|ssq]",
	Short: "生成预测号码",
	Long:  `基于历史数据和AI分析生成选号建议`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lotteryType := types.LotteryType(args[0])
		if lotteryType != types.LotteryTypeDLT && lotteryType != types.LotteryTypeSSQ {
			fmt.Println("错误：彩票类型必须是 dlt(大乐透) 或 ssq(双色球)")
			return
		}

		strategy, _ := cmd.Flags().GetString("strategy")
		count, _ := cmd.Flags().GetInt("count")
		useAI, _ := cmd.Flags().GetBool("ai")

		cfg := config.Get()

		// 初始化存储
		store, err := storage.New(cfg.Database.Path)
		if err != nil {
			fmt.Printf("连接数据库失败: %v\n", err)
			return
		}
		defer store.Close()

		// 创建预测器
		pred := predictor.New(store, cfg)

		// 解析策略
		var predStrategy predictor.PredictionStrategy
		switch strings.ToLower(strategy) {
		case "hot":
			predStrategy = predictor.StrategyHot
		case "cold":
			predStrategy = predictor.StrategyCold
		case "mix":
			predStrategy = predictor.StrategyMix
		case "ai":
			predStrategy = predictor.StrategyAI
			useAI = true
		default:
			predStrategy = predictor.StrategyRandom
		}

		req := &predictor.PredictRequest{
			Type:     lotteryType,
			Strategy: predStrategy,
			Count:    count,
			UseAI:    useAI,
		}

		fmt.Printf("正在生成%s预测...\n", getLotteryName(lotteryType))
		if useAI {
			fmt.Println("使用AI智能分析...")
		}

		predictions, err := pred.Predict(req)
		if err != nil {
			fmt.Printf("预测失败: %v\n", err)
			return
		}

		// 显示结果
		fmt.Printf("\n%s预测结果：\n", getLotteryName(lotteryType))
		fmt.Println(strings.Repeat("=", 50))
		
		for i, p := range predictions {
			fmt.Printf("\n预测 #%d\n", i+1)
			fmt.Printf("策略: %s\n", p.Strategy)
			fmt.Printf("红球: %v\n", p.RedNumbers)
			fmt.Printf("蓝球: %v\n", p.BlueNumbers)
			fmt.Printf("置信度: %.2f%%\n", p.Confidence*100)
			if p.Analysis != "" {
				fmt.Printf("分析: %s\n", p.Analysis)
			}
		}
	},
}

// analyzeCmd 分析命令
var analyzeCmd = &cobra.Command{
	Use:   "analyze [dlt|ssq]",
	Short: "分析历史数据",
	Long:  `分析历史开奖数据，生成冷热号、趋势等统计报告`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lotteryType := types.LotteryType(args[0])
		if lotteryType != types.LotteryTypeDLT && lotteryType != types.LotteryTypeSSQ {
			fmt.Println("错误：彩票类型必须是 dlt(大乐透) 或 ssq(双色球)")
			return
		}

		cfg := config.Get()

		// 初始化存储
		store, err := storage.New(cfg.Database.Path)
		if err != nil {
			fmt.Printf("连接数据库失败: %v\n", err)
			return
		}
		defer store.Close()

		// 创建预测器用于生成分析报告
		pred := predictor.New(store, cfg)

		fmt.Printf("正在分析%s历史数据...\n", getLotteryName(lotteryType))

		report, err := pred.GenerateAnalysisReport(lotteryType)
		if err != nil {
			fmt.Printf("分析失败: %v\n", err)
			return
		}

		// 显示分析报告
		fmt.Printf("\n%s数据分析报告\n", getLotteryName(lotteryType))
		fmt.Println(strings.Repeat("=", 50))

		fmt.Println("\n【红球热号】")
		for _, nf := range report.HotRedNumbers {
			fmt.Printf("  %d号 - 出现%d次\n", nf.Number, nf.Frequency)
		}

		fmt.Println("\n【红球冷号】")
		for _, nf := range report.ColdRedNumbers {
			fmt.Printf("  %d号 - 出现%d次\n", nf.Number, nf.Frequency)
		}

		fmt.Println("\n【蓝球热号】")
		for _, nf := range report.HotBlueNumbers {
			fmt.Printf("  %d号 - 出现%d次\n", nf.Number, nf.Frequency)
		}

		fmt.Println("\n【蓝球冷号】")
		for _, nf := range report.ColdBlueNumbers {
			fmt.Printf("  %d号 - 出现%d次\n", nf.Number, nf.Frequency)
		}

		fmt.Printf("\n【趋势分析】\n  %s\n", report.Trend)
	},
}

// listCmd 列出数据命令
var listCmd = &cobra.Command{
	Use:   "list [dlt|ssq]",
	Short: "列出历史开奖数据",
	Long:  `显示数据库中存储的历史开奖记录`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lotteryType := types.LotteryType(args[0])
		if lotteryType != types.LotteryTypeDLT && lotteryType != types.LotteryTypeSSQ {
			fmt.Println("错误：彩票类型必须是 dlt(大乐透) 或 ssq(双色球)")
			return
		}

		limit, _ := cmd.Flags().GetInt("limit")
		cfg := config.Get()

		store, err := storage.New(cfg.Database.Path)
		if err != nil {
			fmt.Printf("连接数据库失败: %v\n", err)
			return
		}
		defer store.Close()

		results, err := store.GetDrawResults(lotteryType, limit)
		if err != nil {
			fmt.Printf("查询数据失败: %v\n", err)
			return
		}

		fmt.Printf("\n%s历史开奖数据（最近%d期）\n", getLotteryName(lotteryType), len(results))
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("%-12s %-12s %-20s %-15s\n", "期号", "日期", "红球", "蓝球")
		fmt.Println(strings.Repeat("-", 60))

		for _, r := range results {
			fmt.Printf("%-12s %-12s %-20v %-15v\n",
				r.Issue,
				r.DrawDate.Format("2006-01-02"),
				r.RedNumbers,
				r.BlueNumbers,
			)
		}
	},
}

// versionCmd 版本命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()
		fmt.Printf("%s v%s\n", cfg.App.Name, cfg.App.Version)
	},
}

func init() {
	// 添加fetch命令
	fetchCmd.Flags().IntP("limit", "l", 100, "抓取的数据条数")
	rootCmd.AddCommand(fetchCmd)

	// 添加predict命令
	predictCmd.Flags().StringP("strategy", "s", "random", "预测策略: random/hot/cold/mix/ai")
	predictCmd.Flags().IntP("count", "n", 5, "生成预测的数量")
	predictCmd.Flags().BoolP("ai", "a", false, "使用AI分析")
	rootCmd.AddCommand(predictCmd)

	// 添加analyze命令
	rootCmd.AddCommand(analyzeCmd)

	// 添加list命令
	listCmd.Flags().IntP("limit", "l", 20, "显示的数据条数")
	rootCmd.AddCommand(listCmd)

	// 添加version命令
	rootCmd.AddCommand(versionCmd)
}

// getLotteryName 获取彩票名称
func getLotteryName(t types.LotteryType) string {
	switch t {
	case types.LotteryTypeDLT:
		return "大乐透"
	case types.LotteryTypeSSQ:
		return "双色球"
	default:
		return "未知"
	}
}
