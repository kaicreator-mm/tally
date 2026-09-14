package main

import (
	"context"
	"testing"
)

// P0-08:report 命令族分派 —— 缺子命令与未知子命令都显式报错,
// 不静默选择缺省行为。
func TestRunReportDispatch(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"缺子命令", nil},
		{"未知子命令", []string{"monthly"}},
	}
	for _, c := range cases {
		if err := runReport(context.Background(), c.args); err == nil {
			t.Errorf("%s: 应报错,得到 nil", c.name)
		}
	}
}

// P0-08:baseline 不接受位置参数(参数必须在打开数据库之前被拒绝,
// 单元环境无库也不误触连接)。
func TestRunReportBaselineRejectsPositionalArg(t *testing.T) {
	if err := runReportBaseline(context.Background(), []string{"extra"}); err == nil {
		t.Fatal("位置参数应报错")
	}
	if err := runReportBaseline(context.Background(), []string{"-bogus"}); err == nil {
		t.Fatal("未知 flag 应报错")
	}
}
