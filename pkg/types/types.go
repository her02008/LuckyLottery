package types

import "time"

type LotteryType string

const (
	LotteryTypeDLT LotteryType = "dlt"
	LotteryTypeSSQ LotteryType = "ssq"
)

type DrawResult struct {
	ID          int64       `json:"id"`
	Type        LotteryType `json:"type"`
	Issue       string      `json:"issue"`
	DrawDate    time.Time   `json:"draw_date"`
	RedNumbers  []int       `json:"red_numbers"`
	BlueNumbers []int       `json:"blue_numbers"`
	CreatedAt   time.Time   `json:"created_at"`
}

type Prediction struct {
	Type         LotteryType `json:"type"`
	Strategy     string      `json:"strategy"`
	RedNumbers   []int       `json:"red_numbers"`
	BlueNumbers  []int       `json:"blue_numbers"`
	Confidence   float64     `json:"confidence"`
	Analysis     string      `json:"analysis,omitempty"`
	GeneratedAt  time.Time   `json:"generated_at"`
}

type AnalysisReport struct {
	Type           LotteryType       `json:"type"`
	HotRedNumbers  []NumberFrequency `json:"hot_red_numbers"`
	ColdRedNumbers []NumberFrequency `json:"cold_red_numbers"`
	HotBlueNumbers []NumberFrequency `json:"hot_blue_numbers"`
	ColdBlueNumbers []NumberFrequency `json:"cold_blue_numbers"`
	Trend          string            `json:"trend"`
	OmitValues     map[int]int       `json:"omit_values,omitempty"`
	GeneratedAt    time.Time         `json:"generated_at"`
}

type NumberFrequency struct {
	Number    int   `json:"number"`
	Frequency int   `json:"frequency"`
	LastSeen  int64 `json:"last_seen,omitempty"`
}
