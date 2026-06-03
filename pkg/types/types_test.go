package types

import (
	"testing"
	"time"
)

func TestLotteryType(t *testing.T) {
	// 测试彩票类型
	if LotteryTypeDLT != "dlt" {
		t.Errorf("LotteryTypeDLT expected 'dlt', got '%s'", LotteryTypeDLT)
	}

	if LotteryTypeSSQ != "ssq" {
		t.Errorf("LotteryTypeSSQ expected 'ssq', got '%s'", LotteryTypeSSQ)
	}
}

func TestDrawResult(t *testing.T) {
	// 测试开奖结果结构
	result := &DrawResult{
		ID:          1,
		Type:        LotteryTypeDLT,
		Issue:       "2024001",
		DrawDate:    time.Now(),
		RedNumbers:  []int{1, 2, 3, 4, 5},
		BlueNumbers: []int{1, 2},
	}

	if result.Type != LotteryTypeDLT {
		t.Errorf("Expected type DLT, got %s", result.Type)
	}

	if len(result.RedNumbers) != 5 {
		t.Errorf("Expected 5 red numbers, got %d", len(result.RedNumbers))
	}

	if len(result.BlueNumbers) != 2 {
		t.Errorf("Expected 2 blue numbers, got %d", len(result.BlueNumbers))
	}
}

func TestPrediction(t *testing.T) {
	// 测试预测结果结构
	prediction := &Prediction{
		Type:         LotteryTypeSSQ,
		Strategy:     "test",
		RedNumbers:   []int{1, 2, 3, 4, 5, 6},
		BlueNumbers:  []int{1},
		Confidence:   0.8,
		Analysis:     "test analysis",
		GeneratedAt:  time.Now(),
	}

	if prediction.Type != LotteryTypeSSQ {
		t.Errorf("Expected type SSQ, got %s", prediction.Type)
	}

	if prediction.Confidence != 0.8 {
		t.Errorf("Expected confidence 0.8, got %f", prediction.Confidence)
	}
}
