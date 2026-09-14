package environment

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 三种健康检查类型(验收要求至少这三种)。
const (
	CheckTCP        = "tcp"
	CheckHTTP       = "http"
	CheckLogPattern = "log-pattern"
)

// CheckResult 是单项健康检查的结果。
type CheckResult struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail,omitempty"` // 没过时说明原因
	Attempts int    `json:"attempts"`         // WaitHealthy 里试了几轮
}

// HealthReport 是全部健康检查的汇总。OK = 每一项都过。
type HealthReport struct {
	OK      bool          `json:"ok"`
	Results []CheckResult `json:"results"`
}

// LogFetcher 取某服务(ref 为空 = 全部)的运行日志,供 log-pattern 检查匹配。
// docker-compose 场景由 lifecycle 注入(docker compose logs);无 log-pattern 检查时可为 nil。
type LogFetcher func(ctx context.Context, service string) (string, error)

// WaitHealthy 按 wait 的策略轮询所有健康检查:全过立刻返回 OK 报告;
// 总超时到了仍未全过,返回最后一次(未过的)报告,由调用方判定失败。
// ctx 取消则带着最后一次报告返回 ctx.Err()。
func WaitHealthy(ctx context.Context, checks []HealthCheck, wait WaitPolicy, fetchLogs LogFetcher) (HealthReport, error) {
	deadline := time.Now().Add(wait.totalTimeout())
	interval := wait.interval()
	attempts := 0
	for {
		attempts++
		report := runChecks(ctx, checks, fetchLogs, attempts)
		if report.OK {
			return report, nil
		}
		if !time.Now().Before(deadline) {
			return report, nil
		}
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// runChecks 跑一轮全部检查。
func runChecks(ctx context.Context, checks []HealthCheck, fetchLogs LogFetcher, attempts int) HealthReport {
	report := HealthReport{OK: len(checks) > 0}
	for _, c := range checks {
		res := runOne(ctx, c, fetchLogs)
		res.Attempts = attempts
		report.Results = append(report.Results, res)
		if !res.OK {
			report.OK = false
		}
	}
	return report
}

// runOne 跑单项检查。
func runOne(ctx context.Context, c HealthCheck, fetchLogs LogFetcher) CheckResult {
	res := CheckResult{Name: c.Name, Type: c.Type}
	var err error
	switch c.Type {
	case CheckTCP:
		err = checkTCP(ctx, c)
	case CheckHTTP:
		err = checkHTTP(ctx, c)
	case CheckLogPattern:
		err = checkLogPattern(ctx, c, fetchLogs)
	default:
		err = fmt.Errorf("未知健康检查类型 %q", c.Type)
	}
	if err != nil {
		res.Detail = err.Error()
		return res
	}
	res.OK = true
	return res
}

// checkTCP 能建立 TCP 连接即视为健康。
func checkTCP(ctx context.Context, c HealthCheck) error {
	cctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	var d net.Dialer
	conn, err := d.DialContext(cctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp %s 连不上: %w", addr, err)
	}
	_ = conn.Close()
	return nil
}

// checkHTTP GET 目标 URL,校验状态码(与可选的响应体子串)。
func checkHTTP(ctx context.Context, c HealthCheck) error {
	cctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return fmt.Errorf("构造请求 %s: %w", c.URL, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s 请求失败: %w", c.URL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case c.ExpectStatus != 0 && resp.StatusCode != c.ExpectStatus:
		return fmt.Errorf("http %s 状态码 %d,期望 %d", c.URL, resp.StatusCode, c.ExpectStatus)
	case c.ExpectStatus == 0 && (resp.StatusCode < 200 || resp.StatusCode >= 300):
		return fmt.Errorf("http %s 状态码 %d,期望 2xx", c.URL, resp.StatusCode)
	}
	if c.ExpectBody != "" && !strings.Contains(string(body), c.ExpectBody) {
		return fmt.Errorf("http %s 响应体不含 %q", c.URL, c.ExpectBody)
	}
	return nil
}

// checkLogPattern 运行日志里须出现 pattern。
func checkLogPattern(ctx context.Context, c HealthCheck, fetchLogs LogFetcher) error {
	if fetchLogs == nil {
		return fmt.Errorf("log-pattern 检查 %q 无日志来源", c.Name)
	}
	logs, err := fetchLogs(ctx, c.Service)
	if err != nil {
		return fmt.Errorf("取日志失败: %w", err)
	}
	if !strings.Contains(logs, c.Pattern) {
		return fmt.Errorf("日志里没找到 %q", c.Pattern)
	}
	return nil
}
