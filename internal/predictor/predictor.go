package predictor

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"lottery-tool/internal/ai"
	"lottery-tool/internal/analyzer"
	"lottery-tool/internal/config"
	"lottery-tool/internal/storage"
	"lottery-tool/pkg/types"
)

// 全局随机数生成器，使用互斥锁保证线程安全
var (
	globalRand     *rand.Rand
	globalRandMu   sync.Mutex
)

func init() {
	globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// Predictor 预测器
type Predictor struct {
	storage   *storage.Storage
	analyzer  *analyzer.Analyzer
	aiClient  *ai.Client
	config    *config.Config
}

// New 创建预测器
func New(storage *storage.Storage, cfg *config.Config) *Predictor {
	p := &Predictor{
		storage:  storage,
		analyzer: analyzer.New(),
		config:   cfg,
	}

	// 初始化AI客户端
	if cfg.AI.Enabled {
		p.aiClient = ai.NewClient(
			cfg.AI.APIURL,
			cfg.AI.APIKey,
			cfg.AI.Model,
			cfg.AI.Timeout,
		)
	}

	return p
}

// PredictionStrategy 预测策略
type PredictionStrategy int

const (
	StrategyRandom PredictionStrategy = iota
	StrategyHot
	StrategyCold
	StrategyMix
	StrategyAI
)

func (s PredictionStrategy) String() string {
	switch s {
	case StrategyRandom:
		return "随机选号"
	case StrategyHot:
		return "热号策略"
	case StrategyCold:
		return "冷号策略"
	case StrategyMix:
		return "冷热混合"
	case StrategyAI:
		return "AI智能"
	default:
		return "未知策略"
	}
}

// PredictRequest 预测请求
type PredictRequest struct {
	Type      types.LotteryType
	Strategy  PredictionStrategy
	Count     int
	UseAI     bool
}

// Predict 生成预测
func (p *Predictor) Predict(req *PredictRequest) ([]*types.Prediction, error) {
	// 获取历史数据
	history, err := p.storage.GetDrawResults(req.Type, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	if len(history) < 10 {
		return nil, fmt.Errorf("insufficient history data: %d records", len(history))
	}

	// 进行数据分析
	analysis, err := p.analyzer.Analyze(history)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze data: %w", err)
	}

	var predictions []*types.Prediction

	// 如果启用AI且策略为AI，使用AI预测
	if req.UseAI && req.Strategy == StrategyAI && p.aiClient != nil {
		aiPrediction, err := p.aiClient.Predict(req.Type, analysis.ToAnalysisReport(), history)
		if err != nil {
			// AI预测失败，回退到普通策略
			fmt.Printf("AI prediction failed: %v, falling back to standard strategy\n", err)
		} else {
			predictions = append(predictions, aiPrediction)
		}
	}

	// 生成指定数量的预测
	for i := len(predictions); i < req.Count; i++ {
		prediction := p.generateByStrategy(req.Type, req.Strategy, analysis)
		prediction.Strategy = req.Strategy.String()
		predictions = append(predictions, prediction)
	}

	return predictions, nil
}

// generateByStrategy 根据策略生成预测
func (p *Predictor) generateByStrategy(lotteryType types.LotteryType, strategy PredictionStrategy, analysis *analyzer.Analysis) *types.Prediction {
	switch strategy {
	case StrategyHot:
		return p.generateHotStrategy(lotteryType, analysis)
	case StrategyCold:
		return p.generateColdStrategy(lotteryType, analysis)
	case StrategyMix:
		return p.generateMixStrategy(lotteryType, analysis)
	default:
		return p.generateRandomStrategy(lotteryType, analysis)
	}
}

// generateRandomStrategy 随机策略
func (p *Predictor) generateRandomStrategy(lotteryType types.LotteryType, analysis *analyzer.Analysis) *types.Prediction {
	var redCount, blueCount, redMax, blueMax int
	
	switch lotteryType {
	case types.LotteryTypeDLT:
		redCount = 5
		blueCount = 2
		redMax = 35
		blueMax = 12
	case types.LotteryTypeSSQ:
		redCount = 6
		blueCount = 1
		redMax = 33
		blueMax = 16
	}

	prediction := &types.Prediction{
		Type:        lotteryType,
		GeneratedAt: time.Now(),
		Confidence:  0.3,
	}

	// 随机选择红球
	globalRandMu.Lock()
	redSet := make(map[int]bool)
	for len(redSet) < redCount {
		num := globalRand.Intn(redMax) + 1
		if !redSet[num] {
			redSet[num] = true
			prediction.RedNumbers = append(prediction.RedNumbers, num)
		}
	}

	// 随机选择蓝球
	blueSet := make(map[int]bool)
	for len(blueSet) < blueCount {
		num := globalRand.Intn(blueMax) + 1
		if !blueSet[num] {
			blueSet[num] = true
			prediction.BlueNumbers = append(prediction.BlueNumbers, num)
		}
	}
	globalRandMu.Unlock()
	sort.Ints(prediction.RedNumbers)
	sort.Ints(prediction.BlueNumbers)

	return prediction
}

// generateHotStrategy 热号策略
func (p *Predictor) generateHotStrategy(lotteryType types.LotteryType, analysis *analyzer.Analysis) *types.Prediction {
	var redCount, blueCount int
	
	switch lotteryType {
	case types.LotteryTypeDLT:
		redCount = 5
		blueCount = 2
	case types.LotteryTypeSSQ:
		redCount = 6
		blueCount = 1
	}

	prediction := &types.Prediction{
		Type:        lotteryType,
		GeneratedAt: time.Now(),
		Confidence:  0.5,
	}

	// 从热号中选择红球
	hotRedCount := len(analysis.HotRedNumbers)
	if hotRedCount >= redCount {
		for i := 0; i < redCount; i++ {
			prediction.RedNumbers = append(prediction.RedNumbers, analysis.HotRedNumbers[i].Number)
		}
	} else {
		// 热号不够，补充其他号码
		for _, stat := range analysis.HotRedNumbers {
			prediction.RedNumbers = append(prediction.RedNumbers, stat.Number)
		}
		// 从其他号码中随机补充
		p.fillRedBalls(prediction, lotteryType, analysis)
	}
	sort.Ints(prediction.RedNumbers)

	// 从热号中选择蓝球
	hotBlueCount := len(analysis.HotBlueNumbers)
	if hotBlueCount >= blueCount {
		for i := 0; i < blueCount; i++ {
			prediction.BlueNumbers = append(prediction.BlueNumbers, analysis.HotBlueNumbers[i].Number)
		}
	} else {
		for _, stat := range analysis.HotBlueNumbers {
			prediction.BlueNumbers = append(prediction.BlueNumbers, stat.Number)
		}
		p.fillBlueBalls(prediction, lotteryType, analysis)
	}
	sort.Ints(prediction.BlueNumbers)

	return prediction
}

// generateColdStrategy 冷号策略
func (p *Predictor) generateColdStrategy(lotteryType types.LotteryType, analysis *analyzer.Analysis) *types.Prediction {
	var redCount, blueCount int
	
	switch lotteryType {
	case types.LotteryTypeDLT:
		redCount = 5
		blueCount = 2
	case types.LotteryTypeSSQ:
		redCount = 6
		blueCount = 1
	}

	prediction := &types.Prediction{
		Type:        lotteryType,
		GeneratedAt: time.Now(),
		Confidence:  0.4,
	}

	// 从冷号中选择红球
	coldRedCount := len(analysis.ColdRedNumbers)
	if coldRedCount >= redCount {
		for i := 0; i < redCount; i++ {
			prediction.RedNumbers = append(prediction.RedNumbers, analysis.ColdRedNumbers[i].Number)
		}
	} else {
		for _, stat := range analysis.ColdRedNumbers {
			prediction.RedNumbers = append(prediction.RedNumbers, stat.Number)
		}
		p.fillRedBalls(prediction, lotteryType, analysis)
	}
	sort.Ints(prediction.RedNumbers)

	// 从冷号中选择蓝球
	coldBlueCount := len(analysis.ColdBlueNumbers)
	if coldBlueCount >= blueCount {
		for i := 0; i < blueCount; i++ {
			prediction.BlueNumbers = append(prediction.BlueNumbers, analysis.ColdBlueNumbers[i].Number)
		}
	} else {
		for _, stat := range analysis.ColdBlueNumbers {
			prediction.BlueNumbers = append(prediction.BlueNumbers, stat.Number)
		}
		p.fillBlueBalls(prediction, lotteryType, analysis)
	}
	sort.Ints(prediction.BlueNumbers)

	return prediction
}

// generateMixStrategy 冷热混合策略
func (p *Predictor) generateMixStrategy(lotteryType types.LotteryType, analysis *analyzer.Analysis) *types.Prediction {
	var redCount, blueCount int
	
	switch lotteryType {
	case types.LotteryTypeDLT:
		redCount = 5
		blueCount = 2
	case types.LotteryTypeSSQ:
		redCount = 6
		blueCount = 1
	}

	prediction := &types.Prediction{
		Type:        lotteryType,
		GeneratedAt: time.Now(),
		Confidence:  0.45,
	}

	// 60%热号 + 40%冷号
	hotRedCount := redCount * 3 / 5
	coldRedCount := redCount - hotRedCount

	// 选择热号红球
	for i := 0; i < hotRedCount && i < len(analysis.HotRedNumbers); i++ {
		prediction.RedNumbers = append(prediction.RedNumbers, analysis.HotRedNumbers[i].Number)
	}

	// 选择冷号红球
	for i := 0; i < coldRedCount && i < len(analysis.ColdRedNumbers); i++ {
		prediction.RedNumbers = append(prediction.RedNumbers, analysis.ColdRedNumbers[i].Number)
	}

	// 如果数量不够，补充其他号码
	p.fillRedBalls(prediction, lotteryType, analysis)
	sort.Ints(prediction.RedNumbers)

	// 蓝球同样混合
	hotBlueCount := blueCount / 2
	if hotBlueCount == 0 {
		hotBlueCount = 1
	}
	coldBlueCount := blueCount - hotBlueCount

	for i := 0; i < hotBlueCount && i < len(analysis.HotBlueNumbers); i++ {
		prediction.BlueNumbers = append(prediction.BlueNumbers, analysis.HotBlueNumbers[i].Number)
	}

	for i := 0; i < coldBlueCount && i < len(analysis.ColdBlueNumbers); i++ {
		prediction.BlueNumbers = append(prediction.BlueNumbers, analysis.ColdBlueNumbers[i].Number)
	}

	p.fillBlueBalls(prediction, lotteryType, analysis)
	sort.Ints(prediction.BlueNumbers)

	return prediction
}

// fillRedBalls 补充红球
func (p *Predictor) fillRedBalls(prediction *types.Prediction, lotteryType types.LotteryType, analysis *analyzer.Analysis) {
	var targetCount, maxNum int
	
	switch lotteryType {
	case types.LotteryTypeDLT:
		targetCount = 5
		maxNum = 35
	case types.LotteryTypeSSQ:
		targetCount = 6
		maxNum = 33
	}

	existing := make(map[int]bool)
	for _, num := range prediction.RedNumbers {
		existing[num] = true
	}

	globalRandMu.Lock()
	defer globalRandMu.Unlock()
	for len(prediction.RedNumbers) < targetCount {
		num := globalRand.Intn(maxNum) + 1
		if !existing[num] {
			existing[num] = true
			prediction.RedNumbers = append(prediction.RedNumbers, num)
		}
	}
}

// fillBlueBalls 补充蓝球
func (p *Predictor) fillBlueBalls(prediction *types.Prediction, lotteryType types.LotteryType, analysis *analyzer.Analysis) {
	var targetCount, maxNum int
	
	switch lotteryType {
	case types.LotteryTypeDLT:
		targetCount = 2
		maxNum = 12
	case types.LotteryTypeSSQ:
		targetCount = 1
		maxNum = 16
	}

	existing := make(map[int]bool)
	for _, num := range prediction.BlueNumbers {
		existing[num] = true
	}

	globalRandMu.Lock()
	defer globalRandMu.Unlock()
	for len(prediction.BlueNumbers) < targetCount {
		num := globalRand.Intn(maxNum) + 1
		if !existing[num] {
			existing[num] = true
			prediction.BlueNumbers = append(prediction.BlueNumbers, num)
		}
	}
}

// GenerateAnalysisReport 生成分析报告
func (p *Predictor) GenerateAnalysisReport(lotteryType types.LotteryType) (*types.AnalysisReport, error) {
	history, err := p.storage.GetDrawResults(lotteryType, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	if len(history) == 0 {
		return nil, fmt.Errorf("no data available")
	}

	analysis, err := p.analyzer.Analyze(history)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze: %w", err)
	}

	return analysis.ToAnalysisReport(), nil
}
