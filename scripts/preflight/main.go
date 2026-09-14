// A0-03 预检工具入口:分阶段执行检查,输出 tally.preflight.v1 JSON(stdout)
// 与人类可读进度(stderr),退出码见文件头注释。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// schemaVersion 是机器可读报告的结构版本;消费者按此解析。
const schemaVersion = "tally.preflight.v1"

// excerptLimit 限制 JSON 里嵌入的命令输出摘录长度,防止报告爆炸。
const excerptLimit = 2000

// Check 是单项检查结果。Status ∈ PASS | FAIL | SKIP。
type Check struct {
	Name             string     `json:"name"`
	Status           string     `json:"status"`
	Gating           bool       `json:"gating"`
	Detail           string     `json:"detail,omitempty"`
	Command          string     `json:"command,omitempty"`
	Elapsed          string     `json:"elapsed,omitempty"`
	ErrorClass       ErrorClass `json:"error_class,omitempty"`
	IsCodeRegression *bool      `json:"is_code_regression,omitempty"`
	OutputExcerpt    string     `json:"output_excerpt,omitempty"`
}

// Report 是机器可读预检报告。
type Report struct {
	Schema           string  `json:"schema"`
	GeneratedAt      string  `json:"generated_at"`
	GoVersion        string  `json:"go_version,omitempty"`
	GoModDirective   string  `json:"go_mod_directive,omitempty"`
	Status           Status  `json:"status"`
	IsCodeRegression bool    `json:"is_code_regression"`
	Checks           []Check `json:"checks"`
}

type preflight struct {
	ctx        context.Context
	goExe      string
	repoDir    string
	withDB     bool
	report     Report
	logf       func(format string, args ...any)
	stoppedRun bool // 命中 gating 失败后,剩余检查被跳过
}

func main() {
	withDB := flag.Bool("integration", false, "在 DATABASE_URL 可达时附加集成测试基线(go test -tags=integration ./...)")
	timeout := flag.Duration("timeout", 20*time.Minute, "整个预检的整体超时")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	p := &preflight{
		ctx:     ctx,
		withDB:  *withDB,
		repoDir: ".",
		logf: func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "[preflight] "+format+"\n", args...)
		},
	}
	p.report.Schema = schemaVersion
	p.report.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	status := p.run()

	p.report.Status = status
	p.report.IsCodeRegression = IsCodeRegression(status)
	if p.stoppedRun {
		p.append(Check{
			Name:   "remaining",
			Status: "SKIP",
			Gating: false,
			Detail: "NOT_RUN(reason: staged preflight stops at first gating failure; unattributed stages are neither PASS nor regression)",
		})
	}

	out, err := json.MarshalIndent(p.report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[preflight] marshal report: %v\n", err)
		os.Exit(4)
	}
	fmt.Println(string(out))
	os.Exit(ExitCodeFor(status))
}

// run 依次执行各阶段;遇到第一个 gating 失败即停止(分阶段归因的前提)。
func (p *preflight) run() Status {
	if status := p.checkToolchain(); status != "" {
		return status
	}
	if status := p.checkModules(); status != "" {
		return status
	}
	if status := p.checkBuildVet(); status != "" {
		return status
	}
	if status := p.checkUnitBaseline(); status != "" {
		return status
	}
	if p.withDB {
		if status := p.checkIntegration(); status != "" {
			return status
		}
	} else {
		p.append(Check{
			Name:   "integration",
			Status: "SKIP",
			Gating: false,
			Detail: "NOT_RUN(reason: -integration not requested; DB-gated baseline is opt-in)",
		})
	}
	return StatusBaselineOK
}

// checkToolchain 校验 go 工具链存在且版本满足 go.mod 指令;附带记录可选工具可用性。
func (p *preflight) checkToolchain() Status {
	exe, err := exec.LookPath("go")
	if err != nil {
		p.append(Check{Name: "toolchain", Status: "FAIL", Gating: true,
			Detail: "go not found in PATH: " + err.Error()})
		return StatusToolchainBlocked
	}
	p.goExe = exe

	res := runCommand(p.ctx, CommandSpec{Path: exe, Args: []string{"version"}})
	line := ParseGoVersionLine(res.Stdout)
	if res.Err != nil || line == "" {
		p.append(Check{Name: "toolchain", Status: "FAIL", Gating: true,
			Detail: fmt.Sprintf("go version unreadable: err=%v stdout=%q", res.Err, res.Stdout)})
		return StatusToolchainBlocked
	}
	p.report.GoVersion = line

	directive := ""
	if data, err := os.ReadFile(filepath.Join(p.repoDir, "go.mod")); err == nil {
		directive = ParseGoModDirective(string(data))
	}
	p.report.GoModDirective = directive

	if !VersionAtLeast(line, directive) {
		p.append(Check{Name: "toolchain", Status: "FAIL", Gating: true,
			Detail: fmt.Sprintf("toolchain %s older than go.mod directive %s; upgrade Go (environment issue, not a source verdict)", line, directive)})
		return StatusToolchainBlocked
	}

	detail := fmt.Sprintf("go %s satisfies go.mod directive %s; optional tools:", line, orNone(directive))
	for _, tool := range []string{"git", "docker", "golangci-lint"} {
		if _, err := exec.LookPath(tool); err == nil {
			detail += fmt.Sprintf(" %s=found", tool)
		} else {
			detail += fmt.Sprintf(" %s=missing(optional)", tool)
		}
	}
	p.append(Check{Name: "toolchain", Status: "PASS", Gating: true, Detail: detail})

	// 可选工具面只记录,不设门。
	p.append(Check{Name: "tools", Status: "PASS", Gating: false,
		Detail: "informational only; required-for-baseline tool is go itself"})
	return ""
}

// checkModules 用 go list -deps -test ./... 探测模块可用性(含测试依赖),
// 失败时对 stderr 做分类:只有 SOURCE 判回归;网络/状态/未知都归依赖阻塞。
func (p *preflight) checkModules() Status {
	spec := CommandSpec{Path: p.goExe, Args: []string{"list", "-deps", "-test", "./..."}, Dir: p.repoDir}
	res := runCommand(p.ctx, spec)
	if !res.Failed() {
		p.append(Check{Name: "modules", Status: "PASS", Gating: true,
			Command: spec.String(), Elapsed: res.Elapsed,
			Detail: "all module + test dependencies resolvable"})
		return ""
	}
	class := ClassifyModuleStageError(res.Stderr)
	reg := class == ErrClassSource
	status := StatusForModuleStage(class)
	detail := "module-stage failure classified " + string(class)
	if class == ErrClassUnknown {
		detail += "; fail-safe: treated as dependency blocked, needs human review, not claimed as PASS nor as regression without evidence"
	}
	if class == ErrClassDependencyNetwork {
		detail += "; offline dependency failure is classified, NOT reported as code regression"
	}
	p.append(Check{Name: "modules", Status: "FAIL", Gating: true,
		Command: spec.String(), Elapsed: res.Elapsed,
		Detail: detail, ErrorClass: class, IsCodeRegression: &reg,
		OutputExcerpt: excerpt(res.Stderr)})
	return status
}

// checkBuildVet 在模块确认可用后编译 + vet;此阶段失败归因于源码。
func (p *preflight) checkBuildVet() Status {
	for _, spec := range []CommandSpec{
		{Path: p.goExe, Args: []string{"build", "./..."}, Dir: p.repoDir},
		{Path: p.goExe, Args: []string{"vet", "./..."}, Dir: p.repoDir},
	} {
		name := "build"
		if spec.Args[0] == "vet" {
			name = "vet"
		}
		res := runCommand(p.ctx, spec)
		if !res.Failed() {
			p.append(Check{Name: name, Status: "PASS", Gating: true,
				Command: spec.String(), Elapsed: res.Elapsed})
			continue
		}
		reg := true
		p.append(Check{Name: name, Status: "FAIL", Gating: true,
			Command: spec.String(), Elapsed: res.Elapsed,
			Detail:           "source-level failure after modules confirmed available: candidate code regression",
			IsCodeRegression: &reg,
			OutputExcerpt:    excerpt(res.Stdout + res.Stderr)})
		return StatusSourceFailure
	}
	return ""
}

// checkUnitBaseline 复刻 Makefile `test` 目标的单元基线(arch + internal,无外部依赖)。
func (p *preflight) checkUnitBaseline() Status {
	for _, spec := range []CommandSpec{
		{Path: p.goExe, Args: []string{"test", "./test/arch/..."}, Dir: p.repoDir},
		{Path: p.goExe, Args: []string{"test", "./internal/..."}, Dir: p.repoDir},
	} {
		name := "unit_arch"
		if spec.Args[1] == "./internal/..." {
			name = "unit_internal"
		}
		res := runCommand(p.ctx, spec)
		if !res.Failed() {
			p.append(Check{Name: name, Status: "PASS", Gating: true,
				Command: spec.String(), Elapsed: res.Elapsed,
				Detail: lastNonEmptyLine(res.Stdout)})
			continue
		}
		var exitErr *exec.ExitError
		note := ""
		if errors.As(res.Err, &exitErr) {
			note = fmt.Sprintf("exit=%d", exitErr.ExitCode())
		} else {
			note = res.Err.Error() // 超时/无法启动:env 层问题,但同样不判 PASS
		}
		reg := true
		p.append(Check{Name: name, Status: "FAIL", Gating: true,
			Command: spec.String(), Elapsed: res.Elapsed,
			Detail:           "unit test failure after modules confirmed available: candidate code regression (" + note + ")",
			IsCodeRegression: &reg,
			OutputExcerpt:    excerpt(res.Stdout + res.Stderr)})
		return StatusSourceFailure
	}
	return ""
}

// checkIntegration 在 DATABASE_URL 可达时跑迁移 + 集成基线。
// 数据库连不上是环境问题 → SKIP(NOT_RUN),不算源码失败;迁移/测试在
// 连接确认可达之后失败才归因源码。
func (p *preflight) checkIntegration() Status {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		p.append(Check{Name: "integration", Status: "SKIP", Gating: false,
			Detail: "NOT_RUN(reason: -integration requested but DATABASE_URL not set)"})
		return ""
	}

	migrate := CommandSpec{Path: p.goExe, Args: []string{"run", "./cmd/tally", "migrate"}, Dir: p.repoDir}
	res := runCommand(p.ctx, migrate)
	if res.Failed() {
		class := ClassifyModuleStageError(res.Stderr + res.Stdout)
		if class == ErrClassDependencyNetwork {
			p.append(Check{Name: "integration_migrate", Status: "SKIP", Gating: false,
				Command: migrate.String(), Elapsed: res.Elapsed,
				Detail:        "NOT_RUN(reason: database unreachable; environment issue, not a source verdict)",
				ErrorClass:    class,
				OutputExcerpt: excerpt(res.Stdout + res.Stderr)})
			return ""
		}
		reg := true
		p.append(Check{Name: "integration_migrate", Status: "FAIL", Gating: true,
			Command: migrate.String(), Elapsed: res.Elapsed,
			Detail:           "migration failure with database reachable: candidate code regression",
			IsCodeRegression: &reg,
			OutputExcerpt:    excerpt(res.Stdout + res.Stderr)})
		return StatusSourceFailure
	}

	p.append(Check{Name: "integration_migrate", Status: "PASS", Gating: true,
		Command: migrate.String(), Elapsed: res.Elapsed,
		Detail: "migrations applied/idempotent on reachable database"})

	spec := CommandSpec{Path: p.goExe, Args: []string{"test", "-tags=integration", "./..."}, Dir: p.repoDir}
	res = runCommand(p.ctx, spec)
	if !res.Failed() {
		p.append(Check{Name: "integration_test", Status: "PASS", Gating: true,
			Command: spec.String(), Elapsed: res.Elapsed,
			Detail: lastNonEmptyLine(res.Stdout)})
		return ""
	}
	reg := true
	p.append(Check{Name: "integration_test", Status: "FAIL", Gating: true,
		Command: spec.String(), Elapsed: res.Elapsed,
		Detail:           "integration failure after database reachability confirmed: candidate code regression (note: a mid-run DB outage would be misattributed here; documented limitation)",
		IsCodeRegression: &reg,
		OutputExcerpt:    excerpt(res.Stdout + res.Stderr)})
	return StatusSourceFailure
}

func (p *preflight) append(c Check) {
	p.report.Checks = append(p.report.Checks, c)
	if c.Status == "FAIL" && c.Gating {
		p.stoppedRun = true
	}
	p.logf("%-20s %-4s %s", c.Name, c.Status, c.Detail)
}

func excerpt(s string) string {
	if len(s) <= excerptLimit {
		return s
	}
	return s[len(s)-excerptLimit:]
}

func lastNonEmptyLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(no output)"
	}
	lines := strings.Split(s, "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
