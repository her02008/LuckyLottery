package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"lottery-tool/pkg/types"
)

// Storage 存储
type Storage struct {
	db *sql.DB
}

// New 创建存储实例
func New(dbPath string) (*Storage, error) {
	// 自动创建数据目录
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return s, nil
}

// initSchema 初始化数据库表结构
func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS draw_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		issue TEXT NOT NULL,
		draw_date DATETIME NOT NULL,
		red_numbers TEXT NOT NULL,
		blue_numbers TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(type, issue)
	);

	CREATE INDEX IF NOT EXISTS idx_draw_results_type ON draw_results(type);
	CREATE INDEX IF NOT EXISTS idx_draw_results_date ON draw_results(draw_date);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveDrawResult 保存开奖结果
func (s *Storage) SaveDrawResult(result *types.DrawResult) error {
	// 将数字数组转换为JSON字符串
	redJSON, err := json.Marshal(result.RedNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal red numbers: %w", err)
	}

	blueJSON, err := json.Marshal(result.BlueNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal blue numbers: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO draw_results (type, issue, draw_date, red_numbers, blue_numbers, created_at)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		result.Type,
		result.Issue,
		result.DrawDate,
		string(redJSON),
		string(blueJSON),
		time.Now(),
	)

	return err
}

// GetDrawResults 获取开奖结果
func (s *Storage) GetDrawResults(lotteryType types.LotteryType, limit int) ([]*types.DrawResult, error) {
	query := `
	SELECT id, type, issue, draw_date, red_numbers, blue_numbers, created_at
	FROM draw_results
	WHERE type = ?
	ORDER BY draw_date DESC
	LIMIT ?
	`

	rows, err := s.db.Query(query, lotteryType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.DrawResult
	for rows.Next() {
		var r types.DrawResult
		var redStr, blueStr string
		err := rows.Scan(
			&r.ID, &r.Type, &r.Issue, &r.DrawDate,
			&redStr, &blueStr, &r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// 解析JSON字符串为数字数组
		if err := json.Unmarshal([]byte(redStr), &r.RedNumbers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal red numbers: %w", err)
		}

		if err := json.Unmarshal([]byte(blueStr), &r.BlueNumbers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal blue numbers: %w", err)
		}

		results = append(results, &r)
	}

	return results, nil
}

// Close 关闭数据库连接
func (s *Storage) Close() error {
	return s.db.Close()
}
