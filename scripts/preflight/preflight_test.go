// A0-03 预检工具的表驱动测试(Uber Go Style Guide 表驱动模式)。
//
// 覆盖两类证据:
//  1. 纯函数分类矩阵:网络/依赖状态失败绝不能被判为 SOURCE(即代码回归);
//  2. 失败路径端到端:在一个刻意离线(GOPROXY=off)的临时模块里制造
//     模块下载失败,断言整体结论是 DEPENDENCY_BLOCKED 且 is_code_regression=false。
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestClassifyModuleStageError(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   ErrorClass
	}{
		{
			name:   "goproxy off / downloads disabled",
			stderr: "go: example.com/foo/stub@v1.0.0: module example.com/foo/stub: reading https://proxy.golang.org/example.com/foo/stub/@v/list: module lookup disabled by GOPROXY=off",
			want:   ErrClassDependencyNetwork,
		},
		{
			name:   "dial tcp refused",
			stderr: "go: github.com/jackc/pgx/v5@v5.7.1: Get \"https://proxy.golang.org/github.com/jackc/pgx/v5/@v/list\": dial tcp 142.250.107.82:443: connectex: A connection attempt failed because the connected party did not properly respond",
			want:   ErrClassDependencyNetwork,
		},
		{
			name:   "dns no such host",
			stderr: "go: golang.org/x/crypto@v0.28.0: Get \"https://proxy.golang.org/golang.org/x/crypto/@v/list\": dial tcp: lookup proxy.golang.org: no such host",
			want:   ErrClassDependencyNetwork,
		},
		{
			name:   "proxy 410 gone",
			stderr: "go: example.com/ vanished@v1.0.0: reading https://proxy.golang.org/example.com/!vanished/@v/list: 410 Gone",
			want:   ErrClassDependencyNetwork,
		},
		{
			name:   "missing go.sum entry",
			stderr: "go: golang.org/x/tools@v0.21.1: missing go.sum entry; to add it:\n\tgo mod download golang.org/x/tools",
			want:   ErrClassDependencyState,
		},
		{
			name:   "missing go.sum entry with file-position prefix wins over source heuristic (go1.27 observed shape)",
			stderr: "main.go:3:8: missing go.sum entry for module providing package example.com/stub (imported by probe); to add:\n\tgo get probe\n",
			want:   ErrClassDependencyState,
		},
		{
			name:   "checksum mismatch",
			stderr: "go: github.com/go-git/go-git/v5@v5.12.0: verifying module: checksum mismatch\n\tdownloaded: h1:xxx\n\tgo.sum: h1:yyy",
			want:   ErrClassDependencyState,
		},
		{
			name:   "go.mod requires newer toolchain",
			stderr: "go: go.mod requires go >= 1.24 (running go 1.23.3; GOTOOLCHAIN=local)",
			want:   ErrClassDependencyState,
		},
		{
			name:   "download error with importer position prefix stays network (go1.27 observed shape)",
			stderr: "go: downloading example.com/stub v1.0.0\nmain.go:3:8: example.com/stub@v1.0.0: Get \"http://127.0.0.1:9/example.com/stub/@v/v1.0.0.zip\": dial tcp 127.0.0.1:9: connectex: No connection could be made because the target machine actively refused it.\n",
			want:   ErrClassDependencyNetwork,
		},
		{
			name:   "source syntax error",
			stderr: "package tally/internal/contract\n\tinternal/contract/contract.go:42:5: syntax error: unexpected EOF, expected }",
			want:   ErrClassSource,
		},
		{
			name:   "source undefined symbol",
			stderr: "# tally/internal/agentruntime\n\truntimes/fake/fake.go:10:2: undefined: runtime.Capability",
			want:   ErrClassSource,
		},
		{
			name:   "windows path compiler diagnostic",
			stderr: "package tally/internal/intake\n\tC:\\Dev\\Code\\tally\\internal\\intake\\intake.go:88:1: expected declaration, found IDENT",
			want:   ErrClassSource,
		},
		{
			name:   "unknown failure stays unknown (fail-safe)",
			stderr: "go: some completely unrecognized failure",
			want:   ErrClassUnknown,
		},
		{
			name:   "empty stderr stays unknown",
			stderr: "",
			want:   ErrClassUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyModuleStageError(tt.stderr); got != tt.want {
				t.Errorf("ClassifyModuleStageError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModuleStageFailureNeverReportsNetworkOrStateAsRegression(t *testing.T) {
	tests := []struct {
		name           string
		class          ErrorClass
		wantStatus     Status
		wantRegression bool
	}{
		{name: "network", class: ErrClassDependencyNetwork, wantStatus: StatusDependencyBlocked, wantRegression: false},
		{name: "state", class: ErrClassDependencyState, wantStatus: StatusDependencyBlocked, wantRegression: false},
		{name: "unknown fail-safe", class: ErrClassUnknown, wantStatus: StatusDependencyBlocked, wantRegression: false},
		{name: "source is regression", class: ErrClassSource, wantStatus: StatusSourceFailure, wantRegression: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StatusForModuleStage(tt.class); got != tt.wantStatus {
				t.Errorf("StatusForModuleStage(%q) = %q, want %q", tt.class, got, tt.wantStatus)
			}
			if got := IsCodeRegression(StatusForModuleStage(tt.class)); got != tt.wantRegression {
				t.Errorf("IsCodeRegression(StatusForModuleStage(%q)) = %v, want %v", tt.class, got, tt.wantRegression)
			}
		})
	}
}

func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		status Status
		want   int
	}{
		{StatusBaselineOK, 0},
		{StatusSourceFailure, 1},
		{StatusDependencyBlocked, 2},
		{StatusToolchainBlocked, 3},
		{Status("bogus"), 4},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			if got := ExitCodeFor(tt.status); got != tt.want {
				t.Errorf("ExitCodeFor(%q) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		name      string
		toolchain string
		directive string
		want      bool
	}{
		{name: "patch above", toolchain: "go1.27.1", directive: "1.23", want: true},
		{name: "equal", toolchain: "go1.23.0", directive: "1.23", want: true},
		{name: "below", toolchain: "go1.21.5", directive: "1.23", want: false},
		{name: "no directive", toolchain: "go1.21.5", directive: "", want: true},
		{name: "garbage toolchain fails safe", toolchain: "devel", directive: "1.23", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := VersionAtLeast(tt.toolchain, tt.directive); got != tt.want {
				t.Errorf("VersionAtLeast(%q, %q) = %v, want %v", tt.toolchain, tt.directive, got, tt.want)
			}
		})
	}
}

func TestParseGoModDirective(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "tally go.mod shape", content: "module tally\n\ngo 1.23\n\nrequire (\n\tgithub.com/go-chi/chi/v5 v5.1.0\n)\n", want: "1.23"},
		{name: "patched directive", content: "module m\n\ngo 1.23.0\n", want: "1.23.0"},
		{name: "absent directive", content: "module m\n", want: ""},
		{name: "require line is not directive", content: "module m\n\nrequire (\n\tgo.uber.org/multierr v1.11.0 // indirect\n)\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseGoModDirective(tt.content); got != tt.want {
				t.Errorf("ParseGoModDirective() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasGoFileLocation(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  bool
	}{
		{name: "unix diag", field: "internal/contract/contract.go:42:5: syntax error", want: true},
		{name: "windows diag", field: "C:\\Dev\\tally\\intake.go:88:1: expected declaration", want: true},
		{name: "go file mention without position", field: "cannot read contract.go: permission denied", want: false},
		{name: "no go file", field: "dial tcp 127.0.0.1:5432 connect: refused", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasGoFileLocation(tt.field); got != tt.want {
				t.Errorf("hasGoFileLocation(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

// TestModulesStageOfflineDependencyFailure 是验收标准的执行证据:
// 刻意离线时代理不可达的依赖下载失败必须被分类为 DEPENDENCY_NETWORK,
// 且整体结论是 DEPENDENCY_BLOCKED,is_code_regression=false —— 不是代码回归。
// 代理指向本机拒绝端口,不产生任何外呼,离线环境下同样成立。
func TestModulesStageOfflineDependencyFailure(t *testing.T) {
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("SKIP(reason: go toolchain unavailable: %v)", err)
	}

	dir := t.TempDir()
	cache := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module preflight/negative/probe\n\ngo 1.23\n\nrequire example.com/tally-preflight-negative-probe/stub v1.0.0\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nimport _ \"example.com/tally-preflight-negative-probe/stub\"\n\nfunc main() {}\n")

	// 刻意离线:GOFLAGS=-mod=mod 允许解析继续到下载步骤;代理指向本机必然拒绝的
	// 端口,模块缓存指向空目录 —— 下载失败确定发生且完全离线。
	res := runCommand(context.Background(),
		CommandSpec{Path: goExe, Args: []string{"list", "-deps", "./..."}, Dir: dir},
		"GOPROXY=http://127.0.0.1:9", "GOFLAGS=-mod=mod", "GOSUMDB=off", "GOWORK=off", "GOMODCACHE="+cache)

	if !res.Failed() {
		t.Fatalf("expected module-stage failure for refused proxy with uncached module, got success; stderr=%q", res.Stderr)
	}

	class := ClassifyModuleStageError(res.Stderr)
	if class != ErrClassDependencyNetwork {
		t.Errorf("offline dependency failure classified as %q, want %q; stderr=%q", class, ErrClassDependencyNetwork, res.Stderr)
	}
	if got := StatusForModuleStage(class); got != StatusDependencyBlocked {
		t.Errorf("overall status = %q, want %q", got, StatusDependencyBlocked)
	}
	if IsCodeRegression(StatusForModuleStage(class)) {
		t.Errorf("offline dependency failure must not be reported as code regression")
	}
}

// TestModulesStageMissingGoSumEntryIsDependencyState 复现 go1.27 的实际形态:
// 只读模式下缺 go.sum 项的解析失败带 `main.go:3:8:` 前缀,但必须被判为
// DEPENDENCY_STATE,整体 DEPENDENCY_BLOCKED,绝不报告成代码回归。
func TestModulesStageMissingGoSumEntryIsDependencyState(t *testing.T) {
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("SKIP(reason: go toolchain unavailable: %v)", err)
	}

	dir := t.TempDir()
	cache := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module preflight/negative/probe2\n\ngo 1.23\n\nrequire example.com/tally-preflight-negative-probe/stub v1.0.0\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nimport _ \"example.com/tally-preflight-negative-probe/stub\"\n\nfunc main() {}\n")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res := runCommand(ctx,
		CommandSpec{Path: goExe, Args: []string{"list", "-deps", "./..."}, Dir: dir},
		"GOFLAGS=", "GOWORK=off", "GOMODCACHE="+cache)

	if !res.Failed() {
		t.Fatalf("expected module-stage failure for missing go.sum entry, got success; stderr=%q", res.Stderr)
	}

	class := ClassifyModuleStageError(res.Stderr)
	if class != ErrClassDependencyState {
		t.Errorf("missing go.sum failure classified as %q, want %q; stderr=%q", class, ErrClassDependencyState, res.Stderr)
	}
	if got := StatusForModuleStage(class); got != StatusDependencyBlocked {
		t.Errorf("overall status = %q, want %q", got, StatusDependencyBlocked)
	}
	if IsCodeRegression(StatusForModuleStage(class)) {
		t.Errorf("dependency-state failure must not be reported as code regression; stderr=%q", res.Stderr)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
