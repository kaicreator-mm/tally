// Package main 实现 A0-03「Validation and dependency preflight」:
// 一个确定性的预检工具,把「依赖/网络不可用」与「源码测试失败」分类开,
// 绝不把依赖失败报告成代码回归。
//
// 分级判定原理(确定性来源):检查按阶段串行,只有前序阶段通过才进入下一阶段,
// 因此失败可归因到它发生的阶段:
//
//	toolchain → modules(go list)→ build → vet → unit(arch + internal)→ integration(可选,需 DATABASE_URL)
//
// 机器可读状态(schema tally.preflight.v1)与退出码:
//
//	0 BASELINE_OK         基线通过
//	1 SOURCE_FAILURE      模块可用之后发生的编译/vet/测试失败 —— 候选代码回归
//	2 DEPENDENCY_BLOCKED  模块/网络/依赖状态问题或无法分类的模块阶段失败 —— 明确不是代码回归
//	3 TOOLCHAIN_BLOCKED   go 缺失或版本低于 go.mod 指令 —— 明确不是代码回归
//
// 用法:go run ./scripts/preflight [-integration]
package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Status 是整体预检结论。
type Status string

const (
	StatusBaselineOK        Status = "BASELINE_OK"
	StatusSourceFailure     Status = "SOURCE_FAILURE"
	StatusDependencyBlocked Status = "DEPENDENCY_BLOCKED"
	StatusToolchainBlocked  Status = "TOOLCHAIN_BLOCKED"
)

// ExitCodeFor 把整体状态映射为进程退出码(与 JSON 里的 status 一致)。
func ExitCodeFor(s Status) int {
	switch s {
	case StatusBaselineOK:
		return 0
	case StatusSourceFailure:
		return 1
	case StatusDependencyBlocked:
		return 2
	case StatusToolchainBlocked:
		return 3
	default:
		return 4
	}
}

// IsCodeRegression 报告该状态是否构成代码回归判定。
// 只有「模块确认可用之后」的源码级失败才算;依赖/工具链问题永远不算。
// 这直接落实验收标准:offline dependency failure is classified,
// not reported as code regression。
func IsCodeRegression(s Status) bool {
	return s == StatusSourceFailure
}

// ErrorClass 是模块阶段失败的分类。
type ErrorClass string

const (
	// ErrClassNone 无失败。
	ErrClassNone ErrorClass = ""
	// ErrClassDependencyNetwork 模块代理/网络不可达(GOPROXY=off、dial tcp、no such host、410/404 等)。
	ErrClassDependencyNetwork ErrorClass = "DEPENDENCY_NETWORK"
	// ErrClassDependencyState 依赖状态损坏(go.sum 缺项、校验和不匹配、go 指令要求更高工具链等)。
	ErrClassDependencyState ErrorClass = "DEPENDENCY_STATE"
	// ErrClassSource 源码解析/类型错误 —— 候选代码回归。
	ErrClassSource ErrorClass = "SOURCE"
	// ErrClassUnknown 无法识别的失败。按 fail-safe 处理:不判 PASS,也不在无证据时判回归。
	ErrClassUnknown ErrorClass = "UNKNOWN"
)

// networkPatterns 是模块下载/网络类失败的 stderr 特征。
var networkPatterns = []string{
	"module lookup disabled", // GOPROXY=off / GOFLAGS 禁用下载
	"GOPROXY=off",
	"downloads disabled",
	"proxyconnect tcp",
	"dial tcp",
	"no such host",
	"connection refused",
	"connection reset",
	"i/o timeout",
	"TLS handshake timeout",
	"404 Not Found",
	"410 Gone",
	"Get \"https://",
	"Get \"http://",
	"Post \"https://",
	"Post \"http://",
}

// statePatterns 是依赖状态/完整性类失败的 stderr 特征。
var statePatterns = []string{
	"missing go.sum entry",
	"updates to go.sum needed",
	"checksum mismatch",
	"SECURITY ERROR",
	"inconsistent vendoring",
	"go.mod requires go >=", // 工具链版本被 go.mod 指令卡住 —— 环境问题
}

// ClassifyModuleStageError 对 go list(模块阶段)的 stderr 做确定性分类。
//
// 顺序敏感,依据两条观察:
//  1. 依赖状态/网络词汇("missing go.sum entry"、"dial tcp"、"Get \"http..." 等)
//     是高精度证据;模块下载失败还会带上导入方的 `file.go:3:8:` 位置前缀
//     (go1.27 实测形态),所以位置启发式不能先判源码;
//     依赖状态优先网络:两者不冲突,状态词汇更精确。
//  2. 源码判定要求「位置诊断 + 编译器词汇」同时出现,缺一不可;
//     都不命中则 UNKNOWN(fail-safe:既不判 PASS 也无证据不判回归)。
func ClassifyModuleStageError(stderr string) ErrorClass {
	if stderr == "" {
		return ErrClassUnknown
	}
	if containsAny(stderr, statePatterns) {
		return ErrClassDependencyState
	}
	if containsAny(stderr, networkPatterns) {
		return ErrClassDependencyNetwork
	}
	if hasGoFileLocation(stderr) && containsAny(stderr, sourceVocab) {
		return ErrClassSource
	}
	return ErrClassUnknown
}

// sourceVocab 是编译器诊断词汇;单独的位置前缀不构成源码错误证据。
var sourceVocab = []string{
	"syntax error",
	"undefined:",
	"expected declaration",
	"expected package name",
	"expected ';'",
	"redeclared as",
	"redeclared in this block",
	"missing return",
	"declared and not used",
	"imported and not used",
	"cannot use",
	"cannot convert",
	"cannot call non-function",
	"cannot take address",
	"assignment mismatch",
	"too many return values",
	"not enough return values",
	"invalid operation:",
	"incompatible types",
	"build constraints exclude all Go files",
}

// hasGoFileLocation 识别 `path/file.go:123:45:` 形态的编译器诊断(含 Windows 路径)。
func hasGoFileLocation(s string) bool {
	for _, field := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' || r == ' ' || r == '\t' }) {
		if !strings.Contains(field, ".go:") {
			continue
		}
		rest := field[strings.Index(field, ".go:")+len(".go:"):]
		if rest == "" {
			continue
		}
		if rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
	return false
}

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// StatusForModuleStage 把模块阶段结果折叠成整体状态。
// SOURCE 是唯一升级为代码回归的模块阶段分类;UNKNOWN 归入依赖阻塞并显式标记,
// 既不判 PASS 也不在无证据时判回归。
func StatusForModuleStage(class ErrorClass) Status {
	switch class {
	case ErrClassSource:
		return StatusSourceFailure
	default:
		return StatusDependencyBlocked
	}
}

// ParseGoVersionLine 从 `go version go1.27.1 windows/amd64` 提取 `go1.27.1`。
func ParseGoVersionLine(line string) string {
	for _, field := range strings.Fields(line) {
		if strings.HasPrefix(field, "go1.") {
			return field
		}
	}
	return ""
}

// VersionAtLeast 判断 toolchain(如 go1.27.1)是否满足 go.mod 指令(如 1.23)。
// 解析失败按不满足处理(fail-safe,由调用方给出可读原因)。
func VersionAtLeast(toolchain, directive string) bool {
	tmaj, tmin, ok := parseVersion(toolchain)
	if !ok {
		return false
	}
	dmaj, dmin, ok := parseVersion(directive)
	if !ok {
		return true // 无指令(或解析失败)时不设门槛
	}
	return tmaj > dmaj || (tmaj == dmaj && tmin >= dmin)
}

// parseVersion 解析 `go1.27.1` / `1.23` / `1.23.0` 为 (major, minor)。
func parseVersion(v string) (int, int, bool) {
	v = strings.TrimPrefix(v, "go")
	major, rest, ok := strings.Cut(v, ".")
	if !ok {
		return 0, 0, false
	}
	minor := rest
	if dot := strings.Index(minor, "."); dot >= 0 {
		minor = minor[:dot]
	}
	maj, minor2 := atoi(major), atoi(minor)
	if maj < 0 || minor2 < 0 {
		return 0, 0, false
	}
	return maj, minor2, true
}

func atoi(s string) int {
	n := 0
	if s == "" {
		return -1
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// ParseGoModDirective 从 go.mod 内容提取 `go` 指令(如 `1.23`);无指令返回空串。
func ParseGoModDirective(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if rest, found := strings.CutPrefix(line, "go "); found {
			rest = strings.TrimSpace(rest)
			if rest != "" && !strings.ContainsAny(rest, " \t") {
				return rest
			}
		}
	}
	return ""
}

// CommandSpec 描述一条预检命令,便于 JSON 里复现。
type CommandSpec struct {
	Path string   `json:"path"`
	Args []string `json:"args"`
	Dir  string   `json:"dir,omitempty"`
}

func (c CommandSpec) String() string {
	return strings.TrimSpace(c.Path + " " + strings.Join(c.Args, " "))
}

// RunResult 是一条命令的执行结果。
type RunResult struct {
	Spec    CommandSpec
	Stdout  string
	Stderr  string
	Err     error
	Elapsed string
}

// Failed 报告命令是否以非零退出。
func (r RunResult) Failed() bool { return r.Err != nil }

// runCommand 在给定环境覆盖下执行命令并捕获输出。
// 覆盖项追加在 os.Environ() 之后:os/exec 对重复键取最后一个值,覆盖项生效。
func runCommand(ctx context.Context, spec CommandSpec, envOverrides ...string) RunResult {
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	if len(envOverrides) > 0 {
		cmd.Env = append(os.Environ(), envOverrides...)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	return RunResult{
		Spec:    spec,
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		Err:     err,
		Elapsed: time.Since(start).Round(time.Millisecond).String(),
	}
}
