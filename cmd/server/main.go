package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"lottery-tool/internal/analyzer"
	"lottery-tool/internal/config"
	"lottery-tool/internal/predictor"
	"lottery-tool/internal/scraper"
	"lottery-tool/internal/storage"
	"lottery-tool/pkg/types"
)

var (
	configPath string
	serverPort int
	rootCmd    = &cobra.Command{
		Use:   "lottery-server",
		Short: "大乐透和双色球选号工具服务器",
		Long:  `一个基于Go语言开发的彩票API服务器`,
		Run:   runServer,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/config.yaml", "配置文件路径")
	rootCmd.PersistentFlags().IntVarP(&serverPort, "port", "p", 51818, "服务端口")
}

func initConfig() {
	if config.Get() == nil {
		if _, err := config.Load(configPath); err != nil {
			log.Fatalf("加载配置失败: %v", err)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	cfg := config.Get()
	store, err := storage.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer store.Close()

	scraperFactory := scraper.NewScraperFactory(cfg.Scraper.SSQURL, cfg.Scraper.DLTURL)
	pred := predictor.New(store, cfg)

	// 创建 HTTP 多路复用器
	mux := http.NewServeMux()

	// API 路由
	api := &APIHandler{
		store:          store,
		scraperFactory: scraperFactory,
		predictor:      pred,
		config:         cfg,
	}

	mux.HandleFunc("/", api.serveIndex)
	mux.HandleFunc("/api/fetch", api.handleFetch)
	mux.HandleFunc("/api/list", api.handleList)
	mux.HandleFunc("/api/analyze", api.handleAnalyze)
	mux.HandleFunc("/api/predict", api.handlePredict)

	addr := fmt.Sprintf(":%d", serverPort)
	log.Printf("服务器启动于 http://localhost%s", addr)
	log.Printf("访问 http://localhost%s 查看测试页面", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// APIHandler API处理器
type APIHandler struct {
	store          *storage.Storage
	scraperFactory *scraper.ScraperFactory
	predictor      *predictor.Predictor
	config         *config.Config
}

// JSONResponse JSON响应
type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp JSONResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *APIHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

func (h *APIHandler) handleFetch(w http.ResponseWriter, r *http.Request) {
	lotteryType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	if lotteryType == "" {
		lotteryType = "ssq"
	}

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	scraperObj, err := h.scraperFactory.Create(types.LotteryType(lotteryType))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("无效的彩票类型: %s", lotteryType),
		})
		return
	}

	results, err := scraperObj.FetchHistory(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("抓取数据失败: %v", err),
		})
		return
	}

	// 保存到数据库
	saved := 0
	for _, result := range results {
		if err := h.store.SaveDrawResult(result); err == nil {
			saved++
		}
	}

	writeJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Message: fmt.Sprintf("成功抓取并保存 %d 条数据", saved),
		Data: map[string]interface{}{
			"fetched": len(results),
			"saved":   saved,
		},
	})
}

func (h *APIHandler) handleList(w http.ResponseWriter, r *http.Request) {
	lotteryType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	if lotteryType == "" {
		lotteryType = "ssq"
	}

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	results, err := h.store.GetDrawResults(types.LotteryType(lotteryType), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("查询数据失败: %v", err),
		})
		return
	}

	// 转换为 API 响应格式
	data := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		data = append(data, map[string]interface{}{
			"issue":       r.Issue,
			"date":        r.DrawDate.Format("2006-01-02"),
			"red_numbers": r.RedNumbers,
			"blue_number": r.BlueNumbers,
		})
	}

	writeJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Data: map[string]interface{}{
			"type":  lotteryType,
			"total": len(results),
			"data":  data,
		},
	})
}

func (h *APIHandler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	lotteryType := r.URL.Query().Get("type")

	if lotteryType == "" {
		lotteryType = "ssq"
	}

	history, err := h.store.GetDrawResults(types.LotteryType(lotteryType), 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("查询数据失败: %v", err),
		})
		return
	}

	if len(history) < 10 {
		writeJSON(w, http.StatusBadRequest, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("数据不足，需要至少10期数据，当前只有 %d 期", len(history)),
		})
		return
	}

	a := analyzer.New()
	analysis, err := a.Analyze(history)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("分析失败: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Data:    analysis.ToAnalysisReport(),
	})
}

func (h *APIHandler) handlePredict(w http.ResponseWriter, r *http.Request) {
	lotteryType := r.URL.Query().Get("type")
	strategy := r.URL.Query().Get("strategy")
	countStr := r.URL.Query().Get("count")

	if lotteryType == "" {
		lotteryType = "ssq"
	}
	if strategy == "" {
		strategy = "random"
	}

	count := 5
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 20 {
			count = c
		}
	}

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
	default:
		predStrategy = predictor.StrategyRandom
	}

	req := &predictor.PredictRequest{
		Type:     types.LotteryType(lotteryType),
		Strategy: predStrategy,
		Count:    count,
		UseAI:    predStrategy == predictor.StrategyAI,
	}

	predictions, err := h.predictor.Predict(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, JSONResponse{
			Success: false,
			Error:   fmt.Sprintf("预测失败: %v", err),
		})
		return
	}

	// 转换为 API 响应格式
	data := make([]map[string]interface{}, 0, len(predictions))
	for _, p := range predictions {
		data = append(data, map[string]interface{}{
			"strategy":     p.Strategy,
			"red_numbers":  p.RedNumbers,
			"blue_numbers": p.BlueNumbers,
			"confidence":   p.Confidence,
		})
	}

	writeJSON(w, http.StatusOK, JSONResponse{
		Success: true,
		Data: map[string]interface{}{
			"type":        lotteryType,
			"strategy":    strategy,
			"predictions": data,
		},
	})
}

// 嵌入式 HTML 页面
const indexHTML = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>彩票工具 - API 测试</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { text-align: center; color: #333; margin-bottom: 30px; }
        .panel { background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); margin-bottom: 20px; overflow: hidden; }
        .panel-header { background: #4a90d9; color: white; padding: 15px 20px; font-weight: bold; }
        .panel-body { padding: 20px; }
        .form-group { margin-bottom: 15px; }
        label { display: inline-block; width: 100px; font-weight: 500; }
        select, input { padding: 8px 12px; border: 1px solid #ddd; border-radius: 4px; width: 200px; }
        button { background: #4a90d9; color: white; border: none; padding: 10px 24px; border-radius: 4px; cursor: pointer; font-size: 14px; margin-right: 10px; }
        button:hover { background: #357abd; }
        button:active { transform: scale(0.98); }
        .result { background: #f8f8f8; border: 1px solid #e0e0e0; border-radius: 4px; padding: 15px; margin-top: 15px; max-height: 400px; overflow-y: auto; white-space: pre-wrap; font-family: 'Consolas', monospace; font-size: 13px; }
        .success { color: #28a745; }
        .error { color: #dc3545; }
        .api-list { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 15px; }
        .api-item { background: #f8f8f8; padding: 15px; border-radius: 4px; border-left: 4px solid #4a90d9; }
        .api-item h3 { color: #333; margin-bottom: 8px; font-size: 16px; }
        .api-item p { color: #666; font-size: 14px; }
        .api-item code { background: #e8e8e8; padding: 2px 6px; border-radius: 3px; font-size: 12px; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { border: 1px solid #ddd; padding: 10px; text-align: left; }
        th { background: #f0f0f0; font-weight: 600; }
        tr:nth-child(even) { background: #fafafa; }
        .ball-red { color: #e74c3c; font-weight: bold; }
        .ball-blue { color: #3498db; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎰 彩票工具 API 测试页面</h1>

        <div class="panel">
            <div class="panel-header">📡 可用 API 接口</div>
            <div class="panel-body">
                <div class="api-list">
                    <div class="api-item">
                        <h3>抓取数据</h3>
                        <p>从网络抓取历史开奖数据</p>
                        <code>GET /api/fetch?type=ssq&limit=100</code>
                    </div>
                    <div class="api-item">
                        <h3>查看历史</h3>
                        <p>查看数据库中的历史记录</p>
                        <code>GET /api/list?type=ssq&limit=20</code>
                    </div>
                    <div class="api-item">
                        <h3>数据分析</h3>
                        <p>分析历史数据的冷热号</p>
                        <code>GET /api/analyze?type=ssq</code>
                    </div>
                    <div class="api-item">
                        <h3>生成预测</h3>
                        <p>根据策略生成预测号码</p>
                        <code>GET /api/predict?type=ssq&strategy=random&count=5</code>
                    </div>
                </div>
            </div>
        </div>

        <div class="panel">
            <div class="panel-header">📥 抓取数据</div>
            <div class="panel-body">
                <div class="form-group">
                    <label>彩票类型:</label>
                    <select id="fetchType">
                        <option value="ssq">双色球</option>
                        <option value="dlt">大乐透</option>
                    </select>
                    <label style="width:80px;margin-left:20px;">数量:</label>
                    <input type="number" id="fetchLimit" value="100" min="1" max="500">
                    <button onclick="fetchData()">抓取</button>
                </div>
                <div id="fetchResult" class="result"></div>
            </div>
        </div>

        <div class="panel">
            <div class="panel-header">📋 查看历史</div>
            <div class="panel-body">
                <div class="form-group">
                    <label>彩票类型:</label>
                    <select id="listType">
                        <option value="ssq">双色球</option>
                        <option value="dlt">大乐透</option>
                    </select>
                    <label style="width:80px;margin-left:20px;">数量:</label>
                    <input type="number" id="listLimit" value="10" min="1" max="100">
                    <button onclick="listData()">查询</button>
                </div>
                <div id="listResult" class="result"></div>
            </div>
        </div>

        <div class="panel">
            <div class="panel-header">📊 数据分析</div>
            <div class="panel-body">
                <div class="form-group">
                    <label>彩票类型:</label>
                    <select id="analyzeType">
                        <option value="ssq">双色球</option>
                        <option value="dlt">大乐透</option>
                    </select>
                    <button onclick="analyzeData()">分析</button>
                </div>
                <div id="analyzeResult" class="result"></div>
            </div>
        </div>

        <div class="panel">
            <div class="panel-header">🎯 生成预测</div>
            <div class="panel-body">
                <div class="form-group">
                    <label>彩票类型:</label>
                    <select id="predictType">
                        <option value="ssq">双色球</option>
                        <option value="dlt">大乐透</option>
                    </select>
                    <label style="width:80px;margin-left:20px;">策略:</label>
                    <select id="predictStrategy">
                        <option value="random">随机选号</option>
                        <option value="hot">热号策略</option>
                        <option value="cold">冷号策略</option>
                        <option value="mix">冷热混合</option>
                    </select>
                    <label style="width:80px;margin-left:20px;">数量:</label>
                    <input type="number" id="predictCount" value="5" min="1" max="20">
                    <button onclick="predictData()">预测</button>
                </div>
                <div id="predictResult" class="result"></div>
            </div>
        </div>
    </div>

    <script>
        const API_BASE = '';

        async function apiCall(url) {
            const resp = await fetch(API_BASE + url);
            return resp.json();
        }

        async function fetchData() {
            const type = document.getElementById('fetchType').value;
            const limit = document.getElementById('fetchLimit').value;
            const el = document.getElementById('fetchResult');
            el.textContent = '正在抓取...';
            try {
                const data = await apiCall('/api/fetch?type=' + type + '&limit=' + limit);
                if (data.success) {
                    el.innerHTML = '<span class="success">✓ ' + data.message + '</span>';
                } else {
                    el.innerHTML = '<span class="error">✗ ' + data.error + '</span>';
                }
            } catch (e) {
                el.innerHTML = '<span class="error">请求失败: ' + e.message + '</span>';
            }
        }

        async function listData() {
            const type = document.getElementById('listType').value;
            const limit = document.getElementById('listLimit').value;
            const el = document.getElementById('listResult');
            el.textContent = '正在查询...';
            try {
                const data = await apiCall('/api/list?type=' + type + '&limit=' + limit);
                if (data.success) {
                    let html = '<table><tr><th>期号</th><th>日期</th><th>红球</th><th>蓝球</th></tr>';
                    data.data.data.forEach(item => {
                        const reds = item.red_numbers.map(n => '<span class="ball-red">' + String(n).padStart(2, '0') + '</span>').join(' ');
                        const blues = item.blue_number.map(n => '<span class="ball-blue">' + String(n).padStart(2, '0') + '</span>').join(' ');
                        html += '<tr><td>' + item.issue + '</td><td>' + item.date + '</td><td>' + reds + '</td><td>' + blues + '</td></tr>';
                    });
                    html += '</table>';
                    el.innerHTML = html;
                } else {
                    el.innerHTML = '<span class="error">✗ ' + data.error + '</span>';
                }
            } catch (e) {
                el.innerHTML = '<span class="error">请求失败: ' + e.message + '</span>';
            }
        }

        async function analyzeData() {
            const type = document.getElementById('analyzeType').value;
            const el = document.getElementById('analyzeResult');
            el.textContent = '正在分析...';
            try {
                const data = await apiCall('/api/analyze?type=' + type);
                if (data.success) {
                    el.innerHTML = '<span class="success">✓ 分析完成</span>\n\n' + JSON.stringify(data.data, null, 2);
                } else {
                    el.innerHTML = '<span class="error">✗ ' + data.error + '</span>';
                }
            } catch (e) {
                el.innerHTML = '<span class="error">请求失败: ' + e.message + '</span>';
            }
        }

        async function predictData() {
            const type = document.getElementById('predictType').value;
            const strategy = document.getElementById('predictStrategy').value;
            const count = document.getElementById('predictCount').value;
            const el = document.getElementById('predictResult');
            el.textContent = '正在生成预测...';
            try {
                const data = await apiCall('/api/predict?type=' + type + '&strategy=' + strategy + '&count=' + count);
                if (data.success) {
                    let html = '<span class="success">✓ 生成 ' + data.data.predictions.length + ' 组预测号码</span>\n\n';
                    data.data.predictions.forEach((p, i) => {
                        const reds = p.red_numbers.map(n => '<span class="ball-red">' + String(n).padStart(2, '0') + '</span>').join(' ');
                        const blues = p.blue_numbers.map(n => '<span class="ball-blue">' + String(n).padStart(2, '0') + '</span>').join(' ');
                        html += '预测 #' + (i+1) + ' [' + p.strategy + ']\n';
                        html += '红球: ' + reds + '\n';
                        html += '蓝球: ' + blues + '\n\n';
                    });
                    el.innerHTML = html;
                } else {
                    el.innerHTML = '<span class="error">✗ ' + data.error + '</span>';
                }
            } catch (e) {
                el.innerHTML = '<span class="error">请求失败: ' + e.message + '</span>';
            }
        }
    </script>
</body>
</html>
`