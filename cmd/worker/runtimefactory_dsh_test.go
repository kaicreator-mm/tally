// runtimefactory_dsh_test.go 拥有 P4-06 的配置测试:DSH canary 缺省禁用
// (AC:invalid configuration fails before task execution + current Codex
// default unaffected)。
package main

import (
	"strings"
	"testing"
)

// 缺省面:canary 开关缺席 → 构造在任务执行前类型化失败,错误指名启用路径。
func TestDSHBuilderDisabledByDefault(t *testing.T) {
	for _, cfg := range []v13Config{
		{Runtime: "dsh"},                                                              // 全缺省
		{Runtime: "dsh", DSHBin: "/fake/dsh"},                                         // 有 bin 无开关
		{Runtime: "dsh", DSHCanary: true},                                             // 有开关无 bin
		{Runtime: "dsh", DSHCanary: true, DSHBin: "   "},                              // 纯空白 bin 同缺失
		{Runtime: "dsh", DSHCanary: true, DSHBin: "/fake/dsh", DSHModel: "BAD MODEL"}, // overlay 不安全 token
	} {
		rt, err := workerRuntimes.build(cfg)
		if err == nil {
			t.Fatalf("canary 配置 %+#v 应构造失败,得到 %+v", cfg, rt)
		}
		if !strings.Contains(err.Error(), "dsh") && !strings.Contains(err.Error(), "canary") && !strings.Contains(err.Error(), "overlay") {
			t.Fatalf("错误应指向 dsh canary/配置面: %v", err)
		}
	}
}

// 显式 canary 配置(开关 + 二进制 + 合法 model token)可构造,产出 DSH 适配器
// (描述符面);更深的有效性闸(钉版身份/权限枚举)在 Start 处执行 —— 同样
// 先于任何任务执行(无会话、无工作树)。
func TestDSHBuilderConstructsWithExplicitCanary(t *testing.T) {
	rt, err := workerRuntimes.build(v13Config{
		Runtime:   "dsh",
		DSHCanary: true,
		DSHBin:    t.TempDir() + "/dsh", // 二进制存在性归 Start 的钉版身份闸
		DSHModel:  "deepseek-chat",
	})
	if err != nil {
		t.Fatalf("显式 canary 配置应可构造: %v", err)
	}
	if rt == nil {
		t.Fatal("应产出 runtime")
	}
}

// AC:现有 Codex 缺省面不受影响 —— codex 显式配置照常构造,fake 缺省照常。
func TestDSHCanaryDoesNotAffectCodexDefault(t *testing.T) {
	rt, err := workerRuntimes.build(v13Config{Runtime: "codex", CodexBin: "/fake/codex"})
	if err != nil || rt == nil {
		t.Fatalf("codex 构造应不受 canary 影响: %v", err)
	}
	def, err := workerRuntimes.build(v13Config{})
	if err != nil || def == nil {
		t.Fatalf("缺省 fake 构造应不受 canary 影响: %v", err)
	}
	// 未知 runtime 的错误信息现在包含 dsh(登记键序派生)。
	_, err = workerRuntimes.build(v13Config{Runtime: "nope"})
	if err == nil || !strings.Contains(err.Error(), "dsh") {
		t.Fatalf("未知 runtime 错误应列出登记面(含 dsh): %v", err)
	}
}
