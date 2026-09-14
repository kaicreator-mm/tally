package dsh

// patch.go 拥有期望配置 overlay:把 Config.Model / Config.Provider 渲染成
// 一份最小的 --patch YAML(spike 条件 6)。这是期望配置管道,与
// runtimes/codex 的 provider config.toml 播种(ADR-0003)同类:
//
//   - 期望配置只经 config overlay,绝不经 argv:headless 无模型旗标
//     (spike §2.5),`--patch <path>` 是 launcher 观测的 overlay 旗标
//     (spike §2.1,可重复、最高优先级)。
//   - 内容全部由类型化的 Config 字段生成,没有任何文件从 owner ~/.dsh 复制;
//     绝不携带机密(认证走环境变量,isolation.go)。
//   - 字段集合封闭:只有 provider/model 两个期望值。渲染器不发明其它配置面
//     (指令/会话状态结构性不可表达 —— 结构体没有那些字段)。
//   - 零值(两者皆空)= 零 overlay:argv 逐字节等于朴素
//     `--profile headless <task>`,行为与未配置完全一致。
//
// 版本敏感性(spike §5):overlay 按 id 锚定默认配置 dump 观测的
// agent-default-model 插件条目(spike §2.5);patch 合并语义(按 id 覆盖)
// 未做 live 验证,是本骨架的已知不确定面 —— live 表面验证由 env-gated 测试
// 与 P4-04+ 的 conformance canary 承担,版本升级时必须重验。

import (
	"fmt"
	"strings"
)

// patchFlag 是 launcher 观测的 overlay 旗标(spike §2.1)。
const patchFlag = "--patch"

// patchFileName 是每次 run 的临时 overlay 文件名(随 per-run 临时目录生灭)。
const patchFileName = "tally-dsh-patch.yaml"

// modelConfigID 是默认配置 dump 观测的默认模型插件条目 id(spike §2.5)。
// 生成的 overlay 以 id 为锚做最小覆盖。
const modelConfigID = "agent-default-model"

// renderPatchOverlay 渲染期望配置 overlay。provider/model 均空 = 零 overlay
// (空串)。非空值必须通过 validConfigToken(可安全裸写入 YAML 流量值的
// 受限字符集),否则显式报错 —— 不做转义器,直接拒绝可疑取值。
func renderPatchOverlay(provider, model string) (string, error) {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	if provider == "" && model == "" {
		return "", nil
	}
	for _, f := range []struct {
		name  string
		value string
	}{
		{"Provider", provider},
		{"Model", model},
	} {
		if f.value != "" && !validConfigToken(f.value) {
			return "", fmt.Errorf("dsh: Config.%s %q 含 overlay YAML 不安全字符(允许:字母数字与 . _ / -;拒绝可疑取值而不做转义)", f.name, f.value)
		}
	}
	var b strings.Builder
	b.WriteString("- id: " + modelConfigID + "\n")
	b.WriteString("  config:\n")
	if provider != "" {
		b.WriteString("    provider: " + provider + "\n")
	}
	if model != "" {
		b.WriteString("    model: " + model + "\n")
	}
	return b.String(), nil
}

// validConfigToken 报告 s 是否可安全裸写入 YAML 值位:非空,且仅含字母数字与
// . _ / -(观测形态 "deepseek-official" / "deepseek-flash" 的字符集)。
func validConfigToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-', r == '/':
		default:
			return false
		}
	}
	return true
}
