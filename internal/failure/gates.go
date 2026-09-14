package failure

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"tally/internal/command"
	"tally/internal/core"
)

// 四道闸验证修复,全过才能进知识库(PRD 第四部分规则 ②③)。
//
// 执行顺序有讲究,不是按编号:G4 是静态判定(零执行成本),其中「改到验证用文件
// 直接拒绝」是最关键的防线,必须先于一切执行;再 G1(复现不稳就不必浪费后续执行),
// 再 G2(修复后连绿),最后 G3(其余目标)。

// 连跑 5 次而不是 1 次:单次红绿可能是偶发,偶发故障以「已验证修复」身份
// 入库会污染知识库(见 docs/02-invariants.md INV-2)。
const reproRuns = 5

// 哨兵错误:哪一闸挡下的是确定性结论,调用方用 errors.Is 分流(docs/04)。
var (
	ErrNotReproducible  = errors.New("失败无法稳定复现")
	ErrFixTouchesTests  = errors.New("修复改动了验证用的文件")
	ErrFixTouchesSource = errors.New("修复改动了应用源码,暂停等人确认")
	ErrFixNotGreen      = errors.New("修复应用后未连绿五次")
	ErrOthersAffected   = errors.New("修复影响了其他构建目标")
	ErrEmptyFix         = errors.New("修复未改动任何文件")
)

// Proof 是四道闸全部通过后产出的验证证据(实现 core.VerificationProof)。
//
// 字段全部不导出、没有导出的构造函数:failure 之外构造不出带内容的 Proof ——
// 这是铁律 2(未验证不入库)的编译期基础。唯一产生路径是闸门函数;
// test/arch 用 AST 检查字段不导出,单测用反射再查一遍。
type Proof struct {
	at    time.Time
	combo core.VersionCombo
	gates string
}

// VerifiedAt 返回验证通过的时刻。
func (p Proof) VerifiedAt() time.Time { return p.at }

// VerifiedCombo 返回验证当时的版本组合(修复的有效性边界)。
func (p Proof) VerifiedCombo() core.VersionCombo { return p.combo }

// GateRecord 返回四道闸的机器记录。
func (p Proof) GateRecord() string { return p.gates }

// Valid 报告证据是否由闸门真实产出:零值(未经验证)无效。
func (p Proof) Valid() bool { return !p.at.IsZero() }

// FileClass 是 G4 对修复涉及文件的分类。
type FileClass int

const (
	// FileWhitelisted:依赖清单、锁文件、Dockerfile、CI 配置、compose/部署清单、
	// 资源限制、健康检查定义 —— 可以直接改。
	FileWhitelisted FileClass = iota
	// FileSource:应用源码 → 暂停等人确认。
	FileSource
	// FileTest:验证用的文件 → 直接拒绝。
	FileTest
)

var (
	testSegments = map[string]bool{
		"test": true, "tests": true, "testdata": true, "__tests__": true, "__mocks__": true,
		"spec": true, "e2e": true, "cypress": true, "playwright": true, "features": true,
	}
	testSuffixes = []string{
		"_test.go", "_test.py", "_test.rb", "_test.js", "_test.ts",
		".test.js", ".test.ts", ".test.jsx", ".test.tsx",
		".spec.js", ".spec.ts", ".spec.jsx", ".spec.tsx", "_spec.rb",
	}
	whitelistNames = map[string]bool{
		// 依赖清单
		"package.json": true, "go.mod": true, "requirements.txt": true, "requirements.in": true,
		"pyproject.toml": true, "pipfile": true, "cargo.toml": true, "pom.xml": true,
		"build.gradle": true, "build.gradle.kts": true, "gemfile": true, "composer.json": true,
		// 锁文件
		"go.sum": true, "package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
		"poetry.lock": true, "pipfile.lock": true, "cargo.lock": true, "gemfile.lock": true,
		"composer.lock": true,
		// 容器与 CI
		".dockerignore": true, ".gitlab-ci.yml": true, "jenkinsfile": true, ".travis.yml": true,
		// 部署清单(Kamal 的 config/deploy.yml 等)
		"deploy.yml": true, "deploy.yaml": true,
	}
	// 这些目录下的 .yml/.yaml 视为 compose/部署清单、资源限制、健康检查定义。
	deploySegments = map[string]bool{
		"k8s": true, "kubernetes": true, "deploy": true, "helm": true, "charts": true,
		".github": true, ".circleci": true,
	}
)

// ClassifyFile 判定单个路径属于哪一类。验证用文件的判定优先 ——
// 拒绝是最强信号:测试目录里的 docker-compose.yml 也是验证设施,不放行。
func ClassifyFile(p string) FileClass {
	segments := strings.Split(strings.Trim(path.Clean("/"+strings.ReplaceAll(p, "\\", "/")), "/"), "/")
	base := strings.ToLower(segments[len(segments)-1])

	for _, seg := range segments[:len(segments)-1] {
		if testSegments[strings.ToLower(seg)] {
			return FileTest
		}
	}
	for _, suf := range testSuffixes {
		if strings.HasSuffix(base, suf) {
			return FileTest
		}
	}
	if strings.HasPrefix(base, "test_") || strings.HasPrefix(base, "spec_") {
		return FileTest
	}

	if isWhitelisted(segments, base) {
		return FileWhitelisted
	}
	return FileSource
}

// isWhitelisted 是 G4 白名单(PRD 第四部分规则 ③):构建与部署失败的修复面
// 基本就是这几类,所以列得全 —— 这也是只做这两格失败的好处之一。
func isWhitelisted(segments []string, base string) bool {
	if whitelistNames[base] {
		return true
	}
	ext := path.Ext(base)
	switch {
	case strings.HasPrefix(base, "dockerfile"), strings.HasSuffix(base, ".dockerfile"):
		return true
	case (strings.HasPrefix(base, "docker-compose") || strings.HasPrefix(base, "compose")) &&
		(ext == ".yml" || ext == ".yaml"):
		return true
	}
	if ext == ".yml" || ext == ".yaml" {
		for _, seg := range segments[:len(segments)-1] {
			if deploySegments[strings.ToLower(seg)] {
				return true
			}
		}
	}
	// 健康检查定义(脚本或清单)。
	if strings.Contains(base, "healthcheck") || strings.Contains(base, "health-check") {
		return true
	}
	return false
}

// CheckFixScope 是 G4 静态判定,不跑任何东西:
//   - 全部白名单 → nil
//   - 改到验证用的文件 → ErrFixTouchesTests。直接拒绝,不是暂停:
//     不然 LLM 最省事的过关方式就是把检查放松。
//   - 改到应用源码 → ErrFixTouchesSource(暂停等人确认)。
func CheckFixScope(changes []core.FileChange) error {
	if len(changes) == 0 {
		return ErrEmptyFix
	}
	var sources []string
	for _, c := range changes {
		switch ClassifyFile(c.Path) {
		case FileTest:
			return fmt.Errorf("%w: %s", ErrFixTouchesTests, c.Path)
		case FileSource:
			sources = append(sources, c.Path)
		}
	}
	if len(sources) > 0 {
		return fmt.Errorf("%w: %s", ErrFixTouchesSource, strings.Join(sources, ", "))
	}
	return nil
}

// ConfirmRepro 是 G1:连跑 reproRuns 次,全红才算稳定复现。
// 有一次绿了(如 5 红里 4 红 1 绿)就返回 false —— 归类据此判 flaky,不入库。
func ConfirmRepro(ctx context.Context, r Runner, spec RunSpec) (bool, error) {
	spec.Fix = nil // 复现是基线,不带修复
	reds := 0
	for i := 0; i < reproRuns; i++ {
		res, err := r.Run(ctx, spec)
		if err != nil {
			return false, fmt.Errorf("G1 复现第 %d 次: %w", i+1, err)
		}
		if !res.Green {
			reds++
		}
	}
	return reds == reproRuns, nil
}

// VerifyInput 是闸门验证的输入。ReproCommand 是结构化复现命令(工作目录在其
// Cwd 里);Reds 是 G1 的结果(几次红),用于闸门记录;走 VerifyCandidate 时
// G1 已在外面跑过(Handle 的流程:复现不稳就不必调 LLM)。
type VerifyInput struct {
	ReproCommand command.CommandSpec
	Candidate    Candidate
	OtherTargets []OtherTarget
	Combo        core.VersionCombo
	Reds         int
	Now          func() time.Time
}

// VerifyCandidate 跑 G4 → G2 → G3:全过返回证据,否则返回对应哨兵错误、证据零值。
// G1(稳定复现)在调用方先跑 —— 复现都不稳的失败不值得花 LLM 与后续执行。
func VerifyCandidate(ctx context.Context, r Runner, in VerifyInput) (Proof, error) {
	if err := CheckFixScope(in.Candidate.Changes); err != nil {
		return Proof{}, err
	}
	return runG2G3(ctx, r, in)
}

// Verify 一次跑完四道闸:G4(范围)→ G1(复现)→ G2(修复后连绿)→ G3(其余目标)。
// 全过返回证据;任何一闸不过返回对应哨兵错误。
func Verify(ctx context.Context, r Runner, in VerifyInput) (Proof, error) {
	if err := CheckFixScope(in.Candidate.Changes); err != nil {
		return Proof{}, err
	}
	stable, err := ConfirmRepro(ctx, r, RunSpec{Command: in.ReproCommand})
	if err != nil {
		return Proof{}, err
	}
	if !stable {
		return Proof{}, ErrNotReproducible
	}
	in.Reds = reproRuns
	return runG2G3(ctx, r, in)
}

// runG2G3 是 G2(应用修复连跑全绿)+ G3(其余构建目标不受影响)。
// 全程只跑结构化 CommandSpec:G2 跑复现命令,G3 跑各目标自己的构建命令
// (Target 只作标签),命令原样交给执行器,不拆不改。
func runG2G3(ctx context.Context, r Runner, in VerifyInput) (Proof, error) {
	spec := RunSpec{Command: in.ReproCommand, Fix: in.Candidate.Changes}
	greens := 0
	for i := 0; i < reproRuns; i++ {
		res, err := r.Run(ctx, spec)
		if err != nil {
			return Proof{}, fmt.Errorf("G2 验证第 %d 次: %w", i+1, err)
		}
		if res.Green {
			greens++
		}
	}
	if greens != reproRuns {
		return Proof{}, fmt.Errorf("%w: %d/%d 绿", ErrFixNotGreen, greens, reproRuns)
	}

	targetGreen := 0
	for _, tgt := range in.OtherTargets {
		tspec := spec
		tspec.Command = tgt.Command
		tspec.Target = tgt.Target
		res, err := r.Run(ctx, tspec)
		if err != nil {
			return Proof{}, fmt.Errorf("G3 其他目标 %s: %w", tgt.Target, err)
		}
		if !res.Green {
			return Proof{}, fmt.Errorf("%w: %s", ErrOthersAffected, tgt.Target)
		}
		targetGreen++
	}

	now := time.Now
	if in.Now != nil {
		now = in.Now
	}
	record := fmt.Sprintf("g1=%d/%dred g2=%d/%dgreen g3=%d/%dtargets g4=%dfiles-ok",
		in.Reds, reproRuns, greens, reproRuns, targetGreen, len(in.OtherTargets), len(in.Candidate.Changes))
	return Proof{at: now().UTC(), combo: in.Combo, gates: record}, nil
}
