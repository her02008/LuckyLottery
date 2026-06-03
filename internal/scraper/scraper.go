package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"lottery-tool/pkg/types"
)

// Scraper 爬虫接口
type Scraper interface {
	FetchHistory(limit int) ([]*types.DrawResult, error)
	FetchLatest() (*types.DrawResult, error)
}

// HTTPClient HTTP客户端
type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *HTTPClient) Get(url string) ([]byte, error) {
	resp, err := h.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// DLTScraper 大乐透爬虫
type DLTScraper struct {
	client *HTTPClient
	baseURL string
}

func NewDLTScraper(baseURL string) *DLTScraper {
	return &DLTScraper{
		client:  NewHTTPClient(30 * time.Second),
		baseURL: baseURL,
	}
}

func (s *DLTScraper) FetchHistory(limit int) ([]*types.DrawResult, error) {
	// 这里使用中国体彩官网API或其他数据源
	// 示例实现，实际使用时需要替换为真实的数据源
	url := fmt.Sprintf("%s?limit=%d", s.baseURL, limit)
	
	data, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DLT data: %w", err)
	}

	return s.parseJSONData(data)
}

func (s *DLTScraper) FetchLatest() (*types.DrawResult, error) {
	results, err := s.FetchHistory(1)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no data found")
	}
	return results[0], nil
}

func (s *DLTScraper) parseJSONData(data []byte) ([]*types.DrawResult, error) {
	// 解析JSON数据
	// 这里需要根据实际的数据格式进行调整
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var results []*types.DrawResult
	for _, item := range rawData {
		result := &types.DrawResult{
			Type: types.LotteryTypeDLT,
		}
		
		// 解析期号
		if issue, ok := item["issue"].(string); ok {
			result.Issue = issue
		}
		
		// 解析日期
		if dateStr, ok := item["date"].(string); ok {
			date, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				result.DrawDate = date
			}
		}
		
		// 解析红球 (大乐透前区 1-35)
		if redStr, ok := item["red"].(string); ok {
			result.RedNumbers = parseNumbers(redStr)
		}
		
		// 解析蓝球 (大乐透后区 1-12)
		if blueStr, ok := item["blue"].(string); ok {
			result.BlueNumbers = parseNumbers(blueStr)
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

func (s *DLTScraper) parseHTMLData(data []byte) ([]*types.DrawResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var results []*types.DrawResult
	
	// 根据实际HTML结构解析
	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 { // 跳过表头
			return
		}
		
		result := &types.DrawResult{
			Type: types.LotteryTypeDLT,
		}
		
		// 解析表格数据
		cells := s.Find("td")
		if cells.Length() >= 4 {
			result.Issue = strings.TrimSpace(cells.Eq(0).Text())
			
			dateStr := strings.TrimSpace(cells.Eq(1).Text())
			if date, err := time.Parse("2006-01-02", dateStr); err == nil {
				result.DrawDate = date
			}
			
			result.RedNumbers = parseNumbers(cells.Eq(2).Text())
			result.BlueNumbers = parseNumbers(cells.Eq(3).Text())
		}
		
		results = append(results, result)
	})
	
	return results, nil
}

// SSQScraper 双色球爬虫
type SSQScraper struct {
	client  *HTTPClient
	baseURL string
}

func NewSSQScraper(baseURL string) *SSQScraper {
	return &SSQScraper{
		client:  NewHTTPClient(30 * time.Second),
		baseURL: baseURL,
	}
}

func (s *SSQScraper) FetchHistory(limit int) ([]*types.DrawResult, error) {
	url := fmt.Sprintf("%s?limit=%d", s.baseURL, limit)
	
	data, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SSQ data: %w", err)
	}

	return s.parseJSONData(data)
}

func (s *SSQScraper) FetchLatest() (*types.DrawResult, error) {
	results, err := s.FetchHistory(1)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no data found")
	}
	return results[0], nil
}

func (s *SSQScraper) parseJSONData(data []byte) ([]*types.DrawResult, error) {
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var results []*types.DrawResult
	for _, item := range rawData {
		result := &types.DrawResult{
			Type: types.LotteryTypeSSQ,
		}
		
		if issue, ok := item["issue"].(string); ok {
			result.Issue = issue
		}
		
		if dateStr, ok := item["date"].(string); ok {
			date, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				result.DrawDate = date
			}
		}
		
		// 双色球红球 1-33
		if redStr, ok := item["red"].(string); ok {
			result.RedNumbers = parseNumbers(redStr)
		}
		
		// 双色球蓝球 1-16
		if blueStr, ok := item["blue"].(string); ok {
			result.BlueNumbers = parseNumbers(blueStr)
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// parseNumbers 解析数字字符串
func parseNumbers(s string) []int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// 支持多种分隔符
	separators := []string{",", " ", "|", "+", ";", "，", "、"}
	
	for _, sep := range separators {
		if strings.Contains(s, sep) {
			parts := strings.Split(s, sep)
			return convertToInts(parts)
		}
	}
	
	// 如果没有分隔符，尝试按空格分割
	parts := strings.Fields(s)
	return convertToInts(parts)
}

func convertToInts(parts []string) []int {
	var nums []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil {
			nums = append(nums, n)
		}
	}
	return nums
}

// ScraperFactory 爬虫工厂
type ScraperFactory struct {
	dltURL string
	ssqURL string
}

func NewScraperFactory(dltURL, ssqURL string) *ScraperFactory {
	return &ScraperFactory{
		dltURL: dltURL,
		ssqURL: ssqURL,
	}
}

func (f *ScraperFactory) Create(lotteryType types.LotteryType) (Scraper, error) {
	switch lotteryType {
	case types.LotteryTypeDLT:
		return NewDLTScraper(f.dltURL), nil
	case types.LotteryTypeSSQ:
		return NewSSQScraper(f.ssqURL), nil
	default:
		return nil, fmt.Errorf("unsupported lottery type: %s", lotteryType)
	}
}
