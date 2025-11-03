package campaignPerformance

import (
	"fmt"
	"math"
	"strings"
)

// PerformanceData - パフォーマンスデータ
type PerformanceData struct {
	Impressions int64   `json:"impressions"`
	Clicks      int64   `json:"clicks"`
	Conversions int64   `json:"conversions"`
	Purchases   int64   `json:"purchases"`
	Spent       float64 `json:"spent"`
	Budget      float64 `json:"budget"`
}

// CalculatedPerformanceMetrics - 計算済み指標
type CalculatedPerformanceMetrics struct {
	CTR         float64 `json:"ctr"`         // %
	CVR         float64 `json:"cvr"`         // %
	CPC         float64 `json:"cpc"`         // 円
	CPA         float64 `json:"cpa"`         // 円
	BudgetUsage float64 `json:"budgetUsage"` // %
}

// CalculateCTR - CTR（クリック率, %）
func CalculateCTR(impressions, clicks int64) float64 {
	if impressions == 0 {
		return 0
	}
	return roundTo((float64(clicks)/float64(impressions))*100, 2)
}

// CalculateCVR - CVR（コンバージョン率, %）
func CalculateCVR(clicks, conversions int64) float64 {
	if clicks == 0 {
		return 0
	}
	return roundTo((float64(conversions)/float64(clicks))*100, 2)
}

// CalculateCPC - CPC（クリック単価, 円）
func CalculateCPC(spent float64, clicks int64) float64 {
	if clicks == 0 {
		return 0
	}
	return math.Round(spent / float64(clicks))
}

// CalculateCPA - CPA（獲得単価, 円）
func CalculateCPA(spent float64, conversions int64) float64 {
	if conversions == 0 {
		return 0
	}
	return math.Round(spent / float64(conversions))
}

// CalculateBudgetUsage - 予算消化率（%）
func CalculateBudgetUsage(spent, budget float64) float64 {
	if budget == 0 {
		return 0
	}
	return roundTo((spent/budget)*100, 1)
}

// CalculatePerformanceMetrics - まとめて計算
func CalculatePerformanceMetrics(p PerformanceData) CalculatedPerformanceMetrics {
	return CalculatedPerformanceMetrics{
		CTR:         CalculateCTR(p.Impressions, p.Clicks),
		CVR:         CalculateCVR(p.Clicks, p.Conversions),
		CPC:         CalculateCPC(p.Spent, p.Clicks),
		CPA:         CalculateCPA(p.Spent, p.Conversions),
		BudgetUsage: CalculateBudgetUsage(p.Spent, p.Budget),
	}
}

// CalculatePurchaseRate - 購入率（%）
func CalculatePurchaseRate(conversions, purchases int64) float64 {
	if conversions == 0 {
		return 0
	}
	return roundTo((float64(purchases)/float64(conversions))*100, 2)
}

// CalculateROAS - ROAS（倍率）
func CalculateROAS(revenue, spent float64) float64 {
	if spent == 0 {
		return 0
	}
	return roundTo(revenue/spent, 2)
}

// CalculateCPM - CPM（円/1000imp）
func CalculateCPM(spent float64, impressions int64) float64 {
	if impressions == 0 {
		return 0
	}
	return math.Round((spent / float64(impressions)) * 1000)
}

// フォーマット群（TSの formatPerformanceValue 相当）

// FormatPercentage - パーセンテージ表示（末尾に%）
func FormatPercentage(value float64, decimals int) string {
	return fmt.Sprintf("%s%%", FormatDecimal(value, decimals))
}

// FormatCurrency - 通貨表示（¥とカンマ区切り、少数切り捨てではなく四捨五入）
func FormatCurrency(value float64) string {
	return "¥" + FormatNumber(math.Round(value))
}

// FormatNumber - カンマ区切り（整数）
func FormatNumber(value float64) string {
	i := int64(math.Round(value))
	s := fmt.Sprintf("%d", i)
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre == 0 {
		pre = 3
	}
	b.WriteString(s[:pre])
	for i := pre; i < n; i += 3 {
		b.WriteString(",")
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatDecimal - 小数点付き
func FormatDecimal(value float64, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, value)
}

// PreparePerformanceData - パフォーマンスデータを統合
func PreparePerformanceData(
	performance *struct {
		Impressions int64
		Clicks      int64
		Conversions int64
		Purchases   int64
	},
	spent float64,
	budget float64,
) PerformanceData {
	if performance == nil {
		return PerformanceData{
			Impressions: 0,
			Clicks:      0,
			Conversions: 0,
			Purchases:   0,
			Spent:       spent,
			Budget:      budget,
		}
	}
	return PerformanceData{
		Impressions: performance.Impressions,
		Clicks:      performance.Clicks,
		Conversions: performance.Conversions,
		Purchases:   performance.Purchases,
		Spent:       spent,
		Budget:      budget,
	}
}

// GeneratePerformanceSummary - サマリー文字列
func GeneratePerformanceSummary(p PerformanceData) string {
	m := CalculatePerformanceMetrics(p)
	return fmt.Sprintf(
		"インプレッション: %s, クリック数: %s, CTR: %s, CVR: %s, 予算消化率: %s",
		FormatNumber(float64(p.Impressions)),
		FormatNumber(float64(p.Clicks)),
		FormatPercentage(m.CTR, 2),
		FormatPercentage(m.CVR, 2),
		FormatPercentage(m.BudgetUsage, 1),
	)
}

// Benchmarks - 評価用しきい値（任意）
type Benchmarks struct {
	MinCTR *float64 // %
	MinCVR *float64 // %
	MaxCPA *float64 // 円
}

// EvaluationResult - 評価結果
type EvaluationResult struct {
	IsGood          bool     `json:"isGood"`
	Warnings        []string `json:"warnings"`
	Recommendations []string `json:"recommendations"`
}

// EvaluatePerformance - パフォーマンス評価
func EvaluatePerformance(p PerformanceData, b *Benchmarks) EvaluationResult {
	m := CalculatePerformanceMetrics(p)
	warnings := []string{}
	reco := []string{}

	minCTR := 1.0
	minCVR := 2.0
	maxCPA := 5000.0
	if b != nil {
		if b.MinCTR != nil {
			minCTR = *b.MinCTR
		}
		if b.MinCVR != nil {
			minCVR = *b.MinCVR
		}
		if b.MaxCPA != nil {
			maxCPA = *b.MaxCPA
		}
	}

	// CTR
	if m.CTR < minCTR {
		warnings = append(warnings, fmt.Sprintf("CTRが低い (%.2f%% < %.2f%%)", m.CTR, minCTR))
		reco = append(reco, "広告クリエイティブの改善やターゲティングの見直しを検討してください")
	}
	// CVR
	if m.CVR < minCVR {
		warnings = append(warnings, fmt.Sprintf("CVRが低い (%.2f%% < %.2f%%)", m.CVR, minCVR))
		reco = append(reco, "ランディングページの改善やオファーの見直しを検討してください")
	}
	// CPA
	if m.CPA > maxCPA {
		warnings = append(warnings, fmt.Sprintf("CPAが高い (%s > %s)", FormatCurrency(m.CPA), FormatCurrency(maxCPA)))
		reco = append(reco, "予算配分の最適化や入札戦略の見直しを検討してください")
	}
	// 予算消化率
	if m.BudgetUsage > 90 {
		warnings = append(warnings, fmt.Sprintf("予算消化率が高い (%.1f%%)", m.BudgetUsage))
		reco = append(reco, "予算の追加または配信ペースの調整を検討してください")
	} else if m.BudgetUsage < 50 && p.Impressions > 0 {
		warnings = append(warnings, fmt.Sprintf("予算消化率が低い (%.1f%%)", m.BudgetUsage))
		reco = append(reco, "入札額の引き上げやターゲット範囲の拡大を検討してください")
	}

	return EvaluationResult{
		IsGood:          len(warnings) == 0,
		Warnings:        warnings,
		Recommendations: reco,
	}
}

// 内部ユーティリティ
func roundTo(v float64, decimals int) float64 {
	if decimals <= 0 {
		return math.Round(v)
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
