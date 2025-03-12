package main

import (
	"context"
	"testing"
	"time"

	"go-stock/backend/data"
	"go-stock/backend/logger"

	"github.com/stretchr/testify/assert"
)

// @Author spark
// @Date 2025/2/24 9:35
// @Desc
// -----------------------------------------------------------------------------------
func TestIsHKTradingTime(t *testing.T) {
	f := IsHKTradingTime(time.Now())
	t.Log(f)
}

func TestIsUSTradingTime(t *testing.T) {

	date := time.Now()
	hour, minute, _ := date.Clock()
	logger.SugaredLogger.Infof("当前时间: %d:%d", hour, minute)

	t.Log(IsUSTradingTime(time.Now()))
}

// 测试 App 结构体初始化
func TestNewApp(t *testing.T) {
	app := NewApp()
	assert.NotNil(t, app)
	assert.NotNil(t, app.cache)
}

// 测试交易时间判断函数
func TestTradingTime(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "工作日交易时间内（上午）",
			time:     time.Date(2024, 3, 15, 10, 30, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "工作日交易时间内（下午）",
			time:     time.Date(2024, 3, 15, 14, 30, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "工作日非交易时间",
			time:     time.Date(2024, 3, 15, 12, 30, 0, 0, time.Local),
			expected: false,
		},
		{
			name:     "周末时间",
			time:     time.Date(2024, 3, 16, 10, 30, 0, 0, time.Local),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTradingTime(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试港股交易时间判断
func TestHKTradingTime(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "港股开市前竞价时段",
			time:     time.Date(2024, 3, 15, 9, 15, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "港股持续交易时段",
			time:     time.Date(2024, 3, 15, 14, 30, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "港股非交易时间",
			time:     time.Date(2024, 3, 15, 17, 30, 0, 0, time.Local),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHKTradingTime(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试美股交易时间判断
func TestUSTradingTime(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "美股常规交易时段",
			time:     time.Date(2024, 3, 15, 22, 30, 0, 0, time.Local),
			expected: true,
		},
		{
			name:     "美股非交易时间",
			time:     time.Date(2024, 3, 15, 5, 30, 0, 0, time.Local),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUSTradingTime(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试股票信息处理
func TestAddStockFollowData(t *testing.T) {
	follow := data.FollowedStock{
		StockCode:  "SH000001",
		CostPrice:  3000.0,
		Volume:     100,
		Price:      3100.0,
		Sort:       1,
		AlarmPrice: 3200.0,
	}

	stockData := &data.StockInfo{
		Code:     "SH000001",
		Name:     "上证指数",
		Price:    "3150.0",
		PreClose: "3100.0",
		Open:     "3120.0",
		High:     "3180.0",
		Low:      "3090.0",
	}

	addStockFollowData(follow, stockData)

	// 验证计算结果
	assert.NotEqual(t, 0.0, stockData.ChangePercent)
	assert.NotEqual(t, 0.0, stockData.ProfitAmount)
	assert.Equal(t, follow.Sort, stockData.Sort)
}

// 测试消息通知类型
func TestGetMsgType(t *testing.T) {
	tests := []struct {
		name     string
		msgType  int
		expected int
	}{
		{"涨跌报警", 1, 60 * 5},
		{"股价报警", 2, 60 * 30},
		{"成本价报警", 3, 60 * 30},
		{"未知类型", 4, 60 * 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMsgTypeTTL(tt.msgType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock DingDing API 接口
type MockDingDingAPI struct{}

func (m *MockDingDingAPI) SendDingDingMessage(message string) string {
	return "success"
}

// 测试发送钉钉消息
func TestSendDingDingMessage(t *testing.T) {
	app := NewApp()
	ctx := context.Background()
	app.ctx = ctx

	result := app.SendDingDingMessage("测试消息", "SH000001")
	// 由于有缓存机制，第一次应该返回成功
	assert.NotEmpty(t, result)

	// 短时间内重复发送应该返回空字符串
	result = app.SendDingDingMessage("测试消息", "SH000001")
	assert.Empty(t, result)
}

// 测试配置相关功能
func TestConfigOperations(t *testing.T) {
	app := NewApp()

	// 测试获取配置
	config := app.GetConfig()
	assert.NotNil(t, config)

	// 测试更新配置
	settings := &data.Settings{
		RefreshInterval: 5,
	}
	result := app.UpdateConfig(settings)
	assert.NotEmpty(t, result)

	// 验证更新后的配置
	newConfig := app.GetConfig()
	assert.Equal(t, settings.RefreshInterval, newConfig.RefreshInterval)
}
