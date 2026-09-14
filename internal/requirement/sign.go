package requirement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/core"
	"tally/internal/gitwriter"
)

// hashString 用 sha256 算内容哈希,带前缀便于排查。所有追溯边指向它(INV-4)。
func hashString(s string) core.ClauseHash {
	sum := sha256.Sum256([]byte(s))
	return core.ClauseHash("sha256:" + hex.EncodeToString(sum[:]))
}

// Identity 是签署身份。可注入,测试不依赖真实 git 配置。
type Identity struct {
	Name  string
	Email string
	When  time.Time // 零值时用 time.Now
}

// ErrStructuralGap 是签署被结构缺口挡下时的哨兵错误(结构缺口是硬闸)。
var ErrStructuralGap = fmt.Errorf("存在结构缺口,清零后才能签署")

// StructuralGapError 携带具体的结构缺口清单,同时满足 errors.Is(·, ErrStructuralGap)。
type StructuralGapError struct{ Issues []Issue }

func (e *StructuralGapError) Error() string {
	parts := make([]string, 0, len(e.Issues))
	for _, i := range e.Issues {
		parts = append(parts, "  "+i.String())
	}
	return ErrStructuralGap.Error() + ":\n" + strings.Join(parts, "\n")
}

// Unwrap 让 errors.Is(err, ErrStructuralGap) 成立。
func (e *StructuralGapError) Unwrap() error { return ErrStructuralGap }

// SignOptions 是一次签署的输入。
type SignOptions struct {
	RepoPath  string     // git 仓库根
	FileName  string     // 需求文件在仓库内的相对路径(/ 分隔),如 .tally/requirements/x.md
	RawSource string     // 要冻结进 git 的原始 markdown(通常就是被解析的那份)
	Doc       *Document  // 解析后的文档(由 RawSource 解析而来)
	Identity  Identity   // 签署身份,必须给
	Signer    git.Signer // 可注入的签名器;nil → 不签名(结构照建,测试用假签名器)
}

// SignResult 是一次签署的产出。
type SignResult struct {
	ClauseHashes map[string]core.ClauseHash // 每条条款一个内容哈希
	Commit       plumbing.Hash              // git 提交
}

// Sign 冻结需求:产出每条条款的内容哈希,并写一个(签名的)git 提交记录这些哈希。
//
// 结构缺口是硬闸:Lint 还有问题就拒绝签署(返回 StructuralGapError)。
// 签名器经 opts.Signer 注入 —— go-git 的 git.Signer 接口,requirement 不碰 openpgp,
// 也不依赖真实 GPG 配置(INV-6:不把 go-git 的签名依赖变成直接依赖)。
func Sign(opts SignOptions) (*SignResult, error) {
	if opts.Doc == nil {
		return nil, fmt.Errorf("签署:缺少解析后的文档")
	}
	if opts.Identity.Name == "" || opts.Identity.Email == "" {
		return nil, fmt.Errorf("签署:缺少身份(Name/Email)")
	}
	if issues := Lint(opts.Doc); HasStructuralGap(issues) {
		return nil, &StructuralGapError{Issues: issues}
	}

	res := &SignResult{ClauseHashes: map[string]core.ClauseHash{}}
	var hashLines []string
	for _, c := range opts.Doc.Clauses {
		h := ContentHash(c)
		res.ClauseHashes[c.ID] = h
		hashLines = append(hashLines, fmt.Sprintf("clause-hash: %s %s", c.ID, h))
	}

	title := opts.Doc.Title
	if title == "" {
		title = "需求"
	}
	msg := "sign: 冻结" + title + "\n\n" + strings.Join(hashLines, "\n") + "\n"

	src := opts.RawSource
	if src == "" {
		src = Serialize(opts.Doc)
	}
	commit, err := commitFile(opts.RepoPath, opts.FileName, []byte(src), msg, opts.Identity, opts.Signer)
	if err != nil {
		return nil, fmt.Errorf("签署提交: %w", err)
	}
	res.Commit = commit
	return res, nil
}

// Serialize 把文档重新序列化为规范 markdown(冻结时的兜底内容)。
func Serialize(doc *Document) string {
	var b strings.Builder
	if doc.Title != "" {
		fmt.Fprintf(&b, "# %s\n\n", doc.Title)
	}
	for _, c := range doc.Clauses {
		fmt.Fprintf(&b, "## %s", c.ID)
		if c.Title != "" {
			fmt.Fprintf(&b, " %s", c.Title)
		}
		b.WriteString("\n\n")
		fmt.Fprintf(&b, "%s\n\n", c.Statement)
		fmt.Fprintf(&b, "验收: %s\n", c.Mechanism)
		for _, cr := range c.Criteria {
			fmt.Fprintf(&b, "- %s\n", cr)
		}
		b.WriteString("\n")
	}
	b.WriteString("## 非目标\n\n")
	for _, ng := range doc.NonGoals {
		fmt.Fprintf(&b, "- %s\n", ng)
	}
	b.WriteString("\n## 术语表\n\n")
	for _, t := range doc.Glossary {
		fmt.Fprintf(&b, "- %s: %s\n", t.Name, t.Definition)
	}
	return b.String()
}

// commitFile 在 repoPath 仓库里写入 fileName(内容为 content)并创建一个提交。
// identity 提供作者/提交者;signer 非 nil 时对提交签名(go-git 的 git.Signer)。
//
// 无签名提交统一经 internal/gitwriter 单一 Git 写边界(PRD §23.1 Single Git
// Writer、DECISION_LOCKS #5):per-repo 互斥 + 跨进程锁 + 期望基线校验 +
// 原子落盘。HEAD 被并发写入推离基线时拒绝(gitwriter.ErrBaseRevision),
// 绝不叠写到未见过的基线上。本包共享此 helper 的全部 git 写 —— 签署、
// interview.Reject 驳回留痕、SpecDefectSink 提案留痕 —— 因此一并被串行化。
//
// 唯一例外是带签名器的提交:gitwriter 没有提交签名面(CommitRequest 只有
// 身份,无 git.Signer),为保住「签署提交真的带签名」的既有语义(INV-6:
// 注入 git.Signer,提交对象携带 PGPSignature),签名路径保留原 go-git 直写。
// 这是边界尚未覆盖的已知残留,等 gitwriter 具备签名能力后收敛;除此之外
// 本包不再直接操作 worktree/index。
func commitFile(repoPath, fileName string, content []byte, message string, id Identity, signer git.Signer) (plumbing.Hash, error) {
	if signer != nil {
		return commitFileSigned(repoPath, fileName, content, message, id, signer)
	}
	return commitFileGitWriter(repoPath, fileName, content, message, id, "")
}

// commitFileGitWriter 经 gitwriter 写入并提交。expectedBase 是调用方编排该
// 变更时针对的 HEAD;空串 = 以调用时刻读到的 HEAD 为基线(与 ContentStore
// 经 P0-16 迁移后的 Head→Commit 模式一致)。返回新提交哈希。
//
// ErrNoChanges(同内容重复提交)不视为失败:gitwriter 无空提交面,而
// 「重复签署也要留下带哈希的签署记录」由现 HEAD 承载 —— 其提交信息正是
// 同一份 clause-hash 清单、树里正是同一份文件内容,返回它,签署仍可追溯。
func commitFileGitWriter(repoPath, fileName string, content []byte, message string, id Identity, expectedBase string) (plumbing.Hash, error) {
	w, err := gitwriter.New(repoPath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("定位仓库 %s: %w", repoPath, err)
	}
	ctx := context.Background() // 本包公开 API 未携带 ctx(调用方接口不在本任务范围内)
	base := expectedBase
	if base == "" {
		if base, err = w.Head(ctx); err != nil {
			return plumbing.ZeroHash, fmt.Errorf("读取仓库状态: %w", err)
		}
	}
	res, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: base,
		Message:      message,
		Files:        []gitwriter.FileUpdate{{Path: fileName, Content: content}},
		Signature:    &gitwriter.Signature{Name: id.Name, Email: id.Email},
	})
	if err != nil {
		if errors.Is(err, gitwriter.ErrNoChanges) {
			if head, herr := w.Head(ctx); herr == nil && head != "" {
				return plumbing.NewHash(head), nil
			}
		}
		return plumbing.ZeroHash, fmt.Errorf("经 GitWriter 提交: %w", err)
	}
	return plumbing.NewHash(res.Commit), nil
}

// commitFileSigned 是带签名器提交的既有 go-git 路径,逐字保留原行为。
// commitFile 的注释解释了它为何仍在本包直写(gitwriter 无签名面)。
//
// 提交构造是真实发生的:开仓库、写文件、暂存、建提交、签名,全部走 go-git。
func commitFileSigned(repoPath, fileName string, content []byte, message string, id Identity, signer git.Signer) (plumbing.Hash, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("打开仓库 %s: %w", repoPath, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("取工作区: %w", err)
	}

	fsPath := filepath.Join(repoPath, filepath.FromSlash(fileName))
	if err := os.MkdirAll(filepath.Dir(fsPath), 0o755); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("建目录: %w", err)
	}
	if err := os.WriteFile(fsPath, content, 0o644); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("写 %s: %w", fileName, err)
	}
	if _, err := wt.Add(fileName); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("暂存 %s: %w", fileName, err)
	}

	when := id.When
	if when.IsZero() {
		when = time.Now()
	}
	sig := &object.Signature{Name: id.Name, Email: id.Email, When: when}
	hash, err := wt.Commit(message, &git.CommitOptions{
		Author:    sig,
		Committer: sig,
		Signer:    signer,
		// 允许空提交:同一内容重复签署也要留下带哈希的签署记录。
		AllowEmptyCommits: true,
	})
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("创建提交: %w", err)
	}
	return hash, nil
}
