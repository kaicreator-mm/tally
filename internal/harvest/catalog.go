package harvest

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"tally/internal/core"
	"tally/internal/gitwriter"
)

// 每个服务在 catalog/ 下的三个文件。
const (
	definitionFileName = "definition.yaml"
	composeFileName    = "compose.yaml"
	verifiedFileName   = "verified.json"
)

// ErrInvalidProof:写入函数拿到的证据无效(零值/伪造)。铁律 2 的运行时兜底。
var ErrInvalidProof = fmt.Errorf("写入 catalog 需要有效验证证据")

// Identity 是提交身份(仿 requirement.Identity)。提交时刻由 git 自身记录
// (经 gitwriter 单一写边界提交,不再注入 When,也暂不支持提交签名)。
type Identity struct {
	Name  string
	Email string
}

// verifiedRecord 是 verified.json 的结构:验证时间 + 完整版本组合 + 出处。
type verifiedRecord struct {
	VerifiedAt time.Time         `json:"verified_at"`
	Combo      core.VersionCombo `json:"version_combo"`
	Record     string            `json:"record"`
	Source     string            `json:"source,omitempty"`
	Service    string            `json:"service"`
}

// GitCatalog 把通过验证的定义写进 catalog/ 并提交 git。
// 提交统一经 internal/gitwriter 单一写边界(PRD v1.3 §23.1):
// per-repo 串行 + 期望基线校验 + 原子落盘,本类型自己不碰 worktree/index。
type GitCatalog struct {
	RepoPath string // git 仓库根,catalog/ 在其下
	Identity Identity
}

// gitSignature 由 Identity 生成 GitWriter 提交签名;Name/Email 缺省时用本地默认
// (与 gitwriter 的包默认一致),提交永不依赖环境 git 配置。
func (id Identity) gitSignature() *gitwriter.Signature {
	name, email := id.Name, id.Email
	if name == "" {
		name = "tally"
	}
	if email == "" {
		email = "tally@tally.local"
	}
	return &gitwriter.Signature{Name: name, Email: email}
}

// Store 把一份**已验证**的定义写进 catalog/<name>/ 并 git commit。
//
// 签名强制要求 Proof(铁律 2 的编译期基础):Proof 字段/构造都不导出,
// 包外调用者拿不到合法 Proof,只能经 Verify 真拉起获得——未验证的定义进不来。
// 运行时再兜底:proof.Valid() 不过直接拒绝,一个文件都不写。
func (c *GitCatalog) Store(ctx context.Context, def Definition, proof Proof) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !proof.Valid() {
		return fmt.Errorf("%w:证据零值或伪造", ErrInvalidProof)
	}
	if def.Name == "" {
		return fmt.Errorf("写入 catalog:定义缺 name")
	}
	// 再校验一遍健康检查:进 catalog 的定义必须带合法健康检查(INV-2 的另一面)。
	if err := validateChecks(def.HealthChecks); err != nil {
		return fmt.Errorf("写入 catalog:定义不合法: %w", err)
	}

	// compose.yaml:compose 内容。
	compose := []byte(def.Compose)

	// definition.yaml:environment 兼容的定义(compose_file 指向同目录 compose.yaml)。
	out := def
	out.ComposeFile = composeFileName
	out.Compose = "" // 内容单独放 compose.yaml,不重复进 definition.yaml
	out.Source = ""  // 出处只进 verified.json
	y, err := yaml.Marshal(out)
	if err != nil {
		return fmt.Errorf("序列化 definition.yaml: %w", err)
	}

	// verified.json:验证时间 + 完整版本组合 + 出处(Apache-2.0 的「标注已修改/保留出处」)。
	rec := verifiedRecord{
		VerifiedAt: proof.VerifiedAt(),
		Combo:      proof.VerifiedCombo(),
		Record:     proof.GateRecord(),
		Source:     def.Source,
		Service:    def.Name,
	}
	vj, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 verified.json: %w", err)
	}

	// 三个文件一次经 GitWriter 落盘并提交:对调用方原子——任一失败整体回滚,
	// 不留半个服务目录;基线被并发提交抢先推进时报 gitwriter.ErrBaseRevision
	// (可用 errors.Is 识别),绝不静默叠写到未见过的基线上。
	dir := filepath.ToSlash(filepath.Join("catalog", def.Name))
	msg := fmt.Sprintf("harvest: 收录已验证服务定义 %s\n\n验证于 %s;版本组合 %v",
		def.Name, proof.VerifiedAt().UTC().Format(time.RFC3339), proof.VerifiedCombo())
	w, err := gitwriter.New(c.RepoPath)
	if err != nil {
		return fmt.Errorf("定位 catalog 仓库: %w", err)
	}
	base, err := w.Head(ctx)
	if err != nil {
		return fmt.Errorf("读取 catalog 仓库状态: %w", err)
	}
	if _, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: base,
		Message:      msg,
		Files: []gitwriter.FileUpdate{
			{Path: dir + "/" + composeFileName, Content: compose},
			{Path: dir + "/" + definitionFileName, Content: y},
			{Path: dir + "/" + verifiedFileName, Content: append(vj, '\n')},
		},
		Signature: c.Identity.gitSignature(),
	}); err != nil {
		return fmt.Errorf("提交 catalog/%s: %w", def.Name, err)
	}
	return nil
}
