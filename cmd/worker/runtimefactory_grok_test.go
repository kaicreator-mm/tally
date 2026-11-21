// runtimefactory_grok_test.go 拥有 v2.1 provider candidate 的配置测试:
// grok canary 缺省禁用(fail-closed 同 DSH P4-06 纪律;Codex/fake 缺省面
// 不受影响)。
package main

import (
	"strings"
	"testing"
)

// 缺省面:canary 开关缺席 → 构造在任务执行前类型化失败,错误指名启用路径。
func TestGrokBuilderDisabledByDefault(t *testing.T) {
	for _, cfg := range []v13Config{
		{Runtime: "grok"},                                                                      // 全缺省
		{Runtime: "grok", GrokBin: "/fake/grok"},                                               // 有 bin 无开关
		{Runtime: "grok", GrokCanary: true},                                                    // 有开关无 bin
		{Runtime: "grok", GrokCanary: true, GrokBin: "   "},                                    // 纯空白 bin 同缺失
		{Runtime: "grok", GrokCanary: true, GrokBin: "/fake/grok", GrokModel: "   "},           // 纯空白 model
		{Runtime: "grok", GrokCanary: true, GrokBin: "/fake/grok", GrokPermissionMode: "nope"}, // 非法枚举
	} {
		rt, err := workerRuntimes.build(cfg)
		if err == nil {
			t.Fatalf("canary 配置 %+#v 应构造失败,得到 %+v", cfg, rt)
		}
		if !strings.Contains(err.Error(), "grok") && !strings.Contains(err.Error(), "canary") {
			t.Fatalf("错误应指向 grok canary/配置面: %v", err)
		}
	}
}

// 显式 canary 配置(开关 + 二进制 + 合法期望配置)可构造,产出 grok 适配器
// (描述符面);更深的有效性闸(二进制存在性/钉版身份)在 Start 处执行 ——
// 同样先于任何任务执行(无会话、无工作树)。
func TestGrokBuilderConstructsWithExplicitCanary(t *testing.T) {
	rt, err := workerRuntimes.build(v13Config{
		Runtime:    "grok",
		GrokCanary: true,
		GrokBin:    t.TempDir() + "/grok", // 二进制存在性归 Start 的钉版身份闸
		GrokModel:  "grok-4.7",
	})
	if err != nil {
		t.Fatalf("显式 canary 配置应可构造: %v", err)
	}
	if rt == nil {
		t.Fatal("应产出 runtime")
	}
	if d := rt.Descriptor(); d.Name != "grok" {
		t.Fatalf("描述符名应为 grok: %q", d.Name)
	}
}

// AC:现有 Codex 缺省面不受影响 —— codex 显式配置照常构造,fake 缺省照常;
// 未知 runtime 的错误信息按登记键序包含 grok。
func TestGrokCanaryDoesNotAffectCodexDefault(t *testing.T) {
	rt, err := workerRuntimes.build(v13Config{Runtime: "codex", CodexBin: "/fake/codex"})
	if err != nil || rt == nil {
		t.Fatalf("codex 构造应不受 canary 影响: %v", err)
	}
	def, err := workerRuntimes.build(v13Config{})
	if err != nil || def == nil {
		t.Fatalf("缺省 fake 构造应不受 canary 影响: %v", err)
	}
	_, err = workerRuntimes.build(v13Config{Runtime: "nope"})
	if err == nil || !strings.Contains(err.Error(), "grok") {
		t.Fatalf("未知 runtime 错误应列出登记面(含 grok): %v", err)
	}
}
