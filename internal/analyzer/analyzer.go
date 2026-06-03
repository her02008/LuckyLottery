package analyzer

import (
	"fmt"
	"sort"
	"time"

	"lottery-tool/pkg/types"
)

// Analyzer 数据分析器
type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

// Analyze 分析历史数据
type Analysis struct {
	Type           types.LotteryType
	RedStats       map[int]*NumberStat
	BlueStats      map[int]*NumberStat
	HotRedNumbers  []NumberStat
	ColdRedNumbers []NumberStat
	HotBlueNumbers []NumberStat
	ColdBlueNumbers []NumberStat
	OmitValues     map[int]int
	Trend          TrendAnalysis
}

// NumberStat 号码统计
type NumberStat struct {
	Number    int
	Frequency int
	LastSeen  int
	AvgGap    float64
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	RedTrend   string
	BlueTrend  string
	OddEven    map[string]int
	BigSmall   map[string]int
}

// Analyze 分析历史数据
func (a *Analyzer) Analyze(results []*types.DrawResult) (*Analysis, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no data to analyze")
	}

	lotteryType := results[0].Type
	analysis := &Analysis{
		Type:       lotteryType,
		RedStats:   make(map[int]*NumberStat),
		BlueStats:  make(map[int]*NumberStat),
		OmitValues: make(map[int]int),
		Trend: TrendAnalysis{
			OddEven:  make(map[string]int),
			BigSmall: make(map[string]int),
		},
	}

	// 初始化号码统计
	a.initNumberStats(analysis, lotteryType)

	// 统计频率和遗漏值
	a.calculateFrequency(results, analysis)

	// 计算冷热号
	a.calculateHotCold(analysis)

	// 分析趋势
	a.analyzeTrend(results, analysis)

	return analysis, nil
}

// initNumberStats 初始化号码统计
func (a *Analyzer) initNumberStats(analysis *Analysis, lotteryType types.LotteryType) {
	var redMax, blueMax int

	switch lotteryType {
	case types.LotteryTypeDLT:
		redMax = 35
		blueMax = 12
	case types.LotteryTypeSSQ:
		redMax = 33
		blueMax = 16
	default:
		return
	}

	// 初始化红球统计
	for i := 1; i <= redMax; i++ {
		analysis.RedStats[i] = &NumberStat{Number: i}
	}

	// 初始化蓝球统计
	for i := 1; i <= blueMax; i++ {
		analysis.BlueStats[i] = &NumberStat{Number: i}
	}
}

// calculateFrequency 计算频率和遗漏值
func (a *Analyzer) calculateFrequency(results []*types.DrawResult, analysis *Analysis) {
	// 记录每个号码上次出现的位置
	lastSeen := make(map[int]int)

	for i, result := range results {
		// 统计红球
		for _, num := range result.RedNumbers {
			if stat, ok := analysis.RedStats[num]; ok {
				stat.Frequency++
				if lastSeen[num] > 0 {
					gap := i - lastSeen[num]
					stat.AvgGap = (stat.AvgGap*float64(stat.Frequency-1) + float64(gap)) / float64(stat.Frequency)
				}
				lastSeen[num] = i
			}
		}

		// 统计蓝球
		for _, num := range result.BlueNumbers {
			if stat, ok := analysis.BlueStats[num]; ok {
				stat.Frequency++
				if lastSeen[num] > 0 {
					gap := i - lastSeen[num]
					stat.AvgGap = (stat.AvgGap*float64(stat.Frequency-1) + float64(gap)) / float64(stat.Frequency)
				}
				lastSeen[num] = i
			}
		}
	}

	// 计算当前遗漏值
	for num, stat := range analysis.RedStats {
		if lastPos, ok := lastSeen[num]; ok {
			stat.LastSeen = lastPos
			analysis.OmitValues[num] = lastPos
		} else {
			stat.LastSeen = len(results)
			analysis.OmitValues[num] = len(results)
		}
	}

	for num, stat := range analysis.BlueStats {
		if lastPos, ok := lastSeen[num+1000]; ok { // 蓝球偏移1000避免冲突
			stat.LastSeen = lastPos
			analysis.OmitValues[num+1000] = lastPos
		} else {
			stat.LastSeen = len(results)
			analysis.OmitValues[num+1000] = len(results)
		}
	}
}

// calculateHotCold 计算冷热号
func (a *Analyzer) calculateHotCold(analysis *Analysis) {
	// 红球冷热号
	var redStats []NumberStat
	for _, stat := range analysis.RedStats {
		redStats = append(redStats, *stat)
	}

	// 按频率排序
	sort.Slice(redStats, func(i, j int) bool {
		return redStats[i].Frequency > redStats[j].Frequency
	})

	// 前30%为热号，后30%为冷号
	redCount := len(redStats)
	hotCount := redCount * 3 / 10
	coldCount := redCount * 3 / 10

	if hotCount == 0 {
		hotCount = 5
	}
	if coldCount == 0 {
		coldCount = 5
	}

	analysis.HotRedNumbers = redStats[:hotCount]
	analysis.ColdRedNumbers = redStats[redCount-coldCount:]

	// 蓝球冷热号
	var blueStats []NumberStat
	for _, stat := range analysis.BlueStats {
		blueStats = append(blueStats, *stat)
	}

	sort.Slice(blueStats, func(i, j int) bool {
		return blueStats[i].Frequency > blueStats[j].Frequency
	})

	blueCount := len(blueStats)
	hotBlueCount := blueCount / 3
	coldBlueCount := blueCount / 3

	if hotBlueCount == 0 {
		hotBlueCount = 2
	}
	if coldBlueCount == 0 {
		coldBlueCount = 2
	}

	analysis.HotBlueNumbers = blueStats[:hotBlueCount]
	analysis.ColdBlueNumbers = blueStats[blueCount-coldBlueCount:]
}

// analyzeTrend 分析趋势
func (a *Analyzer) analyzeTrend(results []*types.DrawResult, analysis *Analysis) {
	if len(results) < 10 {
		return
	}

	// 分析最近10期的趋势
	recentResults := results[:10]

	redOdd, redEven := 0, 0
	redBig, redSmall := 0, 0
	blueOdd, blueEven := 0, 0
	blueBig, blueSmall := 0, 0

	var redMidpoint, blueMidpoint int
	switch analysis.Type {
	case types.LotteryTypeDLT:
		redMidpoint = 18 // 35/2
		blueMidpoint = 6 // 12/2
	case types.LotteryTypeSSQ:
		redMidpoint = 17 // 33/2
		blueMidpoint = 8 // 16/2
	}

	for _, result := range recentResults {
		// 红球奇偶和大小
		for _, num := range result.RedNumbers {
			if num%2 == 1 {
				redOdd++
			} else {
				redEven++
			}
			if num > redMidpoint {
				redBig++
			} else {
				redSmall++
			}
		}

		// 蓝球奇偶和大小
		for _, num := range result.BlueNumbers {
			if num%2 == 1 {
				blueOdd++
			} else {
				blueEven++
			}
			if num > blueMidpoint {
				blueBig++
			} else {
				blueSmall++
			}
		}
	}

	analysis.Trend.OddEven = map[string]int{
		"red_odd":   redOdd,
		"red_even":  redEven,
		"blue_odd":  blueOdd,
		"blue_even": blueEven,
	}

	analysis.Trend.BigSmall = map[string]int{
		"red_big":    redBig,
		"red_small":  redSmall,
		"blue_big":   blueBig,
		"blue_small": blueSmall,
	}

	// 判断趋势
	if redOdd > redEven {
		analysis.Trend.RedTrend = "奇数偏强"
	} else if redEven > redOdd {
		analysis.Trend.RedTrend = "偶数偏强"
	} else {
		analysis.Trend.RedTrend = "奇偶平衡"
	}

	if blueOdd > blueEven {
		analysis.Trend.BlueTrend = "奇数偏强"
	} else if blueEven > blueOdd {
		analysis.Trend.BlueTrend = "偶数偏强"
	} else {
		analysis.Trend.BlueTrend = "奇偶平衡"
	}
}

// ToAnalysisReport 转换为分析报告
func (a *Analysis) ToAnalysisReport() *types.AnalysisReport {
	report := &types.AnalysisReport{
		Type:            a.Type,
		Trend:           a.Trend.RedTrend + ", " + a.Trend.BlueTrend,
		OmitValues:      a.OmitValues,
		GeneratedAt:     time.Now(),
	}

	// 转换红球统计
	for _, stat := range a.HotRedNumbers {
		report.HotRedNumbers = append(report.HotRedNumbers, types.NumberFrequency{
			Number:    stat.Number,
			Frequency: stat.Frequency,
			LastSeen:  int64(stat.LastSeen),
		})
	}

	for _, stat := range a.ColdRedNumbers {
		report.ColdRedNumbers = append(report.ColdRedNumbers, types.NumberFrequency{
			Number:    stat.Number,
			Frequency: stat.Frequency,
			LastSeen:  int64(stat.LastSeen),
		})
	}

	// 转换蓝球统计
	for _, stat := range a.HotBlueNumbers {
		report.HotBlueNumbers = append(report.HotBlueNumbers, types.NumberFrequency{
			Number:    stat.Number,
			Frequency: stat.Frequency,
			LastSeen:  int64(stat.LastSeen),
		})
	}

	for _, stat := range a.ColdBlueNumbers {
		report.ColdBlueNumbers = append(report.ColdBlueNumbers, types.NumberFrequency{
			Number:    stat.Number,
			Frequency: stat.Frequency,
			LastSeen:  int64(stat.LastSeen),
		})
	}

	return report
}
