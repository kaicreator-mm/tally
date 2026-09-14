package environment

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	if err := checkTCP(context.Background(), HealthCheck{Type: CheckTCP, Host: "127.0.0.1", Port: port}); err != nil {
		t.Errorf("监听中的端口应健康: %v", err)
	}

	ln.Close() // 关掉,这个端口现在连不上
	if err := checkTCP(context.Background(), HealthCheck{Type: CheckTCP, Host: "127.0.0.1", Port: port, Timeout: duration(200 * time.Millisecond)}); err == nil {
		t.Error("连不上的端口应不健康")
	}
}

func TestCheckHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "all ok here")
	}))
	defer srv.Close()

	base := HealthCheck{Type: CheckHTTP, URL: srv.URL, Timeout: duration(2 * time.Second)}

	if err := checkHTTP(context.Background(), base); err != nil {
		t.Errorf("200 应健康: %v", err)
	}
	withBody := base
	withBody.ExpectBody = "ok here"
	if err := checkHTTP(context.Background(), withBody); err != nil {
		t.Errorf("响应体含子串应健康: %v", err)
	}
	badBody := base
	badBody.ExpectBody = "not-present"
	if err := checkHTTP(context.Background(), badBody); err == nil {
		t.Error("响应体不含子串应不健康")
	}
	badStatus := base
	badStatus.ExpectStatus = http.StatusTeapot
	if err := checkHTTP(context.Background(), badStatus); err == nil {
		t.Error("状态码不符应不健康")
	}
}

func TestCheckLogPattern(t *testing.T) {
	fetcher := func(_ context.Context, _ string) (string, error) {
		return "boot line\nready to accept connections\n", nil
	}
	hc := HealthCheck{Type: CheckLogPattern, Pattern: "ready to accept"}
	if err := checkLogPattern(context.Background(), hc, fetcher); err != nil {
		t.Errorf("日志含 pattern 应健康: %v", err)
	}
	miss := hc
	miss.Pattern = "not-in-logs"
	if err := checkLogPattern(context.Background(), miss, fetcher); err == nil {
		t.Error("日志不含 pattern 应不健康")
	}
	if err := checkLogPattern(context.Background(), hc, nil); err == nil {
		t.Error("无日志来源应不健康")
	}
}

// 验收要求「含等待重试策略」:一开始没好,几轮后才好,WaitHealthy 应重试到好为止。
func TestWaitHealthyRetries(t *testing.T) {
	calls := 0
	fetcher := func(_ context.Context, _ string) (string, error) {
		calls++
		if calls < 3 {
			return "starting...", nil
		}
		return "ready now", nil
	}
	checks := []HealthCheck{{Name: "log", Type: CheckLogPattern, Pattern: "ready"}}
	wait := WaitPolicy{Timeout: duration(3 * time.Second), Interval: duration(20 * time.Millisecond)}

	report, err := WaitHealthy(context.Background(), checks, wait, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Errorf("重试后应健康: %+v", report)
	}
	if calls < 3 {
		t.Errorf("应重试多轮才健康,实际 %d 轮", calls)
	}
}

func TestWaitHealthyTimesOut(t *testing.T) {
	fetcher := func(_ context.Context, _ string) (string, error) { return "still booting", nil }
	checks := []HealthCheck{{Name: "log", Type: CheckLogPattern, Pattern: "ready"}}
	wait := WaitPolicy{Timeout: duration(80 * time.Millisecond), Interval: duration(20 * time.Millisecond)}

	report, err := WaitHealthy(context.Background(), checks, wait, fetcher)
	if err != nil {
		t.Fatalf("超时不该返回 error(由调用方判定失败): %v", err)
	}
	if report.OK {
		t.Error("一直不好,报告应是不健康")
	}
}
