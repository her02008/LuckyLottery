package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"lottery-tool/pkg/types"
)

// Client AI客户端
type Client struct {
	apiURL  string
	apiKey  string
	model   string
	client  *http.Client
}

// NewClient 创建AI客户端
func NewClient(apiURL, apiKey, model string, timeout int) *Client {
	if timeout == 0 {
		timeout = 30
	}
	return &Client{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Predict 使用AI进行预测
func (c *Client) Predict(lotteryType types.LotteryType, analysis *types.AnalysisReport, history []*types.DrawResult) (*types.Prediction, error) {
	prompt := c.buildPrompt(lotteryType, analysis, history)
	
	reqBody := ChatRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "你是一个专业的彩票分析专家，擅长根据历史数据和统计分析进行预测。请基于提供的数据给出合理的选号建议。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.apiURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	// 解析AI返回的预测结果
	prediction, err := c.parsePrediction(chatResp.Choices[0].Message.Content, lotteryType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prediction: %w", err)
	}

	prediction.Strategy = "AI智能分析"
	prediction.Analysis = chatResp.Choices[0].Message.Content
	
	return prediction, nil
}

// buildPrompt 构建预测提示词
func (c *Client) buildPrompt(lotteryType types.LotteryType, analysis *types.AnalysisReport, history []*types.DrawResult) string {
	var lotteryName string
	var redCount, blueCount int
	var redMax, blueMax int

	switch lotteryType {
	case types.LotteryTypeDLT:
		lotteryName = "大乐透"
		redCount = 5
		blueCount = 2
		redMax = 35
		blueMax = 12
	case types.LotteryTypeSSQ:
		lotteryName = "双色球"
		redCount = 6
		blueCount = 1
		redMax = 33
		blueMax = 16
	}

	prompt := fmt.Sprintf(`请为%s预测下一期的中奖号码。

规则说明：
- 红球范围：1-%d，选择%d个
- 蓝球范围：1-%d，选择%d个

历史数据分析：
`, lotteryName, redMax, redCount, blueMax, blueCount)

	// 添加冷热号信息
	prompt += "\n热号（出现频率高）：\n"
	prompt += "红球热号："
	for _, nf := range analysis.HotRedNumbers {
		prompt += fmt.Sprintf("%d(%d次) ", nf.Number, nf.Frequency)
	}
	prompt += "\n蓝球热号："
	for _, nf := range analysis.HotBlueNumbers {
		prompt += fmt.Sprintf("%d(%d次) ", nf.Number, nf.Frequency)
	}

	prompt += "\n\n冷号（出现频率低）：\n"
	prompt += "红球冷号："
	for _, nf := range analysis.ColdRedNumbers {
		prompt += fmt.Sprintf("%d(%d次) ", nf.Number, nf.Frequency)
	}
	prompt += "\n蓝球冷号："
	for _, nf := range analysis.ColdBlueNumbers {
		prompt += fmt.Sprintf("%d(%d次) ", nf.Number, nf.Frequency)
	}

	prompt += fmt.Sprintf("\n\n趋势分析：%s", analysis.Trend)

	// 添加最近几期开奖结果
	if len(history) > 0 {
		prompt += "\n\n最近开奖结果：\n"
		limit := 5
		if len(history) < limit {
			limit = len(history)
		}
		for i := 0; i < limit; i++ {
			result := history[i]
			prompt += fmt.Sprintf("期号%s: 红球%v 蓝球%v\n", 
				result.Issue, result.RedNumbers, result.BlueNumbers)
		}
	}

	prompt += fmt.Sprintf(`

请基于以上数据分析，给出你的预测。

请按以下格式返回：
红球预测：数字1,数字2,数字3,数字4,数字5%s
蓝球预测：数字1%s
预测理由：简要说明你的分析逻辑

注意：
1. 红球必须是1-%d之间的不同数字
2. 蓝球必须是1-%d之间的不同数字
3. 请给出合理的分析和预测`, 
		func() string {
			if lotteryType == types.LotteryTypeSSQ {
				return ",数字6"
			}
			return ""
		}(),
		func() string {
			if lotteryType == types.LotteryTypeDLT {
				return ",数字2"
			}
			return ""
		}(),
		redMax, blueMax)

	return prompt
}

// parsePrediction 解析AI返回的预测结果
func (c *Client) parsePrediction(content string, lotteryType types.LotteryType) (*types.Prediction, error) {
	prediction := &types.Prediction{
		Type:        lotteryType,
		GeneratedAt: time.Now(),
	}

	// 简单的解析逻辑，实际使用时可能需要更复杂的解析
	// 这里假设AI按照要求的格式返回
	
	// 解析红球
	redStart := -1
	redEnd := -1
	if idx := bytes.Index([]byte(content), []byte("红球预测：")); idx != -1 {
		redStart = idx + len("红球预测：")
		if newline := bytes.Index([]byte(content[redStart:]), []byte("\n")); newline != -1 {
			redEnd = redStart + newline
		}
	}

	if redStart > 0 && redEnd > redStart {
		redStr := content[redStart:redEnd]
		prediction.RedNumbers = parseNumbersFromString(redStr)
	}

	// 解析蓝球
	blueStart := -1
	blueEnd := -1
	if idx := bytes.Index([]byte(content), []byte("蓝球预测：")); idx != -1 {
		blueStart = idx + len("蓝球预测：")
		if newline := bytes.Index([]byte(content[blueStart:]), []byte("\n")); newline != -1 {
			blueEnd = blueStart + newline
		}
	}

	if blueStart > 0 && blueEnd > blueStart {
		blueStr := content[blueStart:blueEnd]
		prediction.BlueNumbers = parseNumbersFromString(blueStr)
	}

	// 验证预测结果
	if err := c.validatePrediction(prediction, lotteryType); err != nil {
		return nil, err
	}

	prediction.Confidence = 0.7 // 默认置信度

	return prediction, nil
}

// validatePrediction 验证预测结果
func (c *Client) validatePrediction(prediction *types.Prediction, lotteryType types.LotteryType) error {
	var expectedRed, expectedBlue, redMax, blueMax int

	switch lotteryType {
	case types.LotteryTypeDLT:
		expectedRed = 5
		expectedBlue = 2
		redMax = 35
		blueMax = 12
	case types.LotteryTypeSSQ:
		expectedRed = 6
		expectedBlue = 1
		redMax = 33
		blueMax = 16
	default:
		return fmt.Errorf("unsupported lottery type")
	}

	// 验证红球数量
	if len(prediction.RedNumbers) != expectedRed {
		return fmt.Errorf("red ball count mismatch: expected %d, got %d", expectedRed, len(prediction.RedNumbers))
	}

	// 验证蓝球数量
	if len(prediction.BlueNumbers) != expectedBlue {
		return fmt.Errorf("blue ball count mismatch: expected %d, got %d", expectedBlue, len(prediction.BlueNumbers))
	}

	// 验证红球范围
	redSet := make(map[int]bool)
	for _, num := range prediction.RedNumbers {
		if num < 1 || num > redMax {
			return fmt.Errorf("red ball out of range: %d", num)
		}
		if redSet[num] {
			return fmt.Errorf("duplicate red ball: %d", num)
		}
		redSet[num] = true
	}

	// 验证蓝球范围
	blueSet := make(map[int]bool)
	for _, num := range prediction.BlueNumbers {
		if num < 1 || num > blueMax {
			return fmt.Errorf("blue ball out of range: %d", num)
		}
		if blueSet[num] {
			return fmt.Errorf("duplicate blue ball: %d", num)
		}
		blueSet[num] = true
	}

	return nil
}

// parseNumbersFromString 从字符串解析数字
func parseNumbersFromString(s string) []int {
	var nums []int
	var current int
	var hasCurrent bool

	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			current = current*10 + int(ch-'0')
			hasCurrent = true
		} else if hasCurrent {
			nums = append(nums, current)
			current = 0
			hasCurrent = false
		}
	}

	if hasCurrent {
		nums = append(nums, current)
	}

	return nums
}
