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
	// 使用内置API或用户配置的API
	if s.baseURL == "" {
		s.baseURL = BuiltInSSQLotteryAPI
	}
	
	// 构造API请求URL（大乐透类型为dlt）
	url := fmt.Sprintf("%s?type=dlt&limit=%d", s.baseURL, limit)
	
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
	// 适配 huiniao.top API 格式
	var response struct {
		Code int `json:"code"`
		Info string `json:"info"`
		Data struct {
			Data struct {
				List []struct {
					Code string `json:"code"` // 期号
					Day  string `json:"day"`  // 日期
					One  string `json:"one"`  // 红球1
					Two  string `json:"two"`  // 红球2
					Three string `json:"three"` // 红球3
					Four  string `json:"four"`  // 红球4
					Five  string `json:"five"`  // 红球5
					Six   string `json:"six"`   // 红球6
					Seven string `json:"seven"` // 蓝球
				} `json:"list"`
			} `json:"data"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	if response.Code != 1 {
		return nil, fmt.Errorf("API error: %s", response.Info)
	}

	var results []*types.DrawResult
	for _, item := range response.Data.Data.List {
		result := &types.DrawResult{
			Type:  types.LotteryTypeDLT,
			Issue: item.Code,
		}
		
		// 解析日期
		if date, err := time.Parse("2006-01-02", item.Day); err == nil {
			result.DrawDate = date
		}
		
		// 解析红球/前区 (one-five)
		result.RedNumbers = []int{}
		for _, n := range []string{item.One, item.Two, item.Three, item.Four, item.Five} {
			if num, err := strconv.Atoi(n); err == nil {
				result.RedNumbers = append(result.RedNumbers, num)
			}
		}
		
		// 解析蓝球/后区 (six-seven)
		result.BlueNumbers = []int{}
		for _, n := range []string{item.Six, item.Seven} {
			if num, err := strconv.Atoi(n); err == nil {
				result.BlueNumbers = append(result.BlueNumbers, num)
			}
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
	// 使用内置API或用户配置的API
	if s.baseURL == "" {
		s.baseURL = BuiltInSSQLotteryAPI
	}
	
	// 构造API请求URL
	url := fmt.Sprintf("%s?type=ssq&limit=%d", s.baseURL, limit)
	
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
	// 适配 huiniao.top API 格式
	var response struct {
		Code int `json:"code"`
		Info string `json:"info"`
		Data struct {
			Data struct {
				List []struct {
					Code string `json:"code"` // 期号
					Day  string `json:"day"`  // 日期
					One  string `json:"one"`  // 红球1
					Two  string `json:"two"`  // 红球2
					Three string `json:"three"` // 红球3
					Four  string `json:"four"`  // 红球4
					Five  string `json:"five"`  // 红球5
					Six   string `json:"six"`   // 红球6
					Seven string `json:"seven"` // 蓝球
				} `json:"list"`
			} `json:"data"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	if response.Code != 1 {
		return nil, fmt.Errorf("API error: %s", response.Info)
	}

	var results []*types.DrawResult
	for _, item := range response.Data.Data.List {
		result := &types.DrawResult{
			Type:  types.LotteryTypeSSQ,
			Issue: item.Code,
		}
		
		// 解析日期
		if date, err := time.Parse("2006-01-02", item.Day); err == nil {
			result.DrawDate = date
		}
		
		// 解析红球 (one-six)
		result.RedNumbers = []int{}
		for _, n := range []string{item.One, item.Two, item.Three, item.Four, item.Five, item.Six} {
			if num, err := strconv.Atoi(n); err == nil {
				result.RedNumbers = append(result.RedNumbers, num)
			}
		}
		
		// 解析蓝球 (seven)
		if num, err := strconv.Atoi(item.Seven); err == nil {
			result.BlueNumbers = []int{num}
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

// 内置免费API（当用户未配置时使用）
const (
	// huiniao.top 免费API
	BuiltInSSQLotteryAPI = "http://api.huiniao.top/interface/home/lotteryHistory"
)

// ScraperFactory 爬虫工厂
type ScraperFactory struct {
	dltURL string
	ssqURL string
}

func NewScraperFactory(dltURL, ssqURL string) *ScraperFactory {
	// 如果用户未配置URL，使用内置免费API
	if ssqURL == "" {
		ssqURL = BuiltInSSQLotteryAPI
	}
	if dltURL == "" {
		dltURL = BuiltInSSQLotteryAPI
	}
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
