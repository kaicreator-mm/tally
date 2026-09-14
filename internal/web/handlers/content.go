package handlers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"

	"tally/internal/gitwriter"
)

// ErrNotFound 表示请求的内容文件不存在。
var ErrNotFound = errors.New("内容不存在")

// ContentStore 是 git 仓库里的用户内容(项目/需求/文档/笔记)。
// 写 = 落文件 + commit(版本/diff/回滚由 git 管);读 = 读工作区文件。
// 所有提交统一经 internal/gitwriter 单一写边界(PRD v1.3 §23.1):
// per-repo 串行 + 期望基线校验,ContentStore 自己不碰 worktree/index。
type ContentStore struct {
	RepoPath string
	Identity Identity
}

// gitSignature 由 Identity 生成 GitWriter 提交签名;Name/Email 缺省时用本地默认。
// Identity.Now 不再参与:提交时刻由 gitwriter/git 以当前时间记录。
func (id Identity) gitSignature() *gitwriter.Signature {
	name, email := id.Name, id.Email
	if name == "" {
		name = "tally"
	}
	if email == "" {
		email = "tally@localhost"
	}
	return &gitwriter.Signature{Name: name, Email: email}
}

// EnsureRepo 确保 path 是一个 git 仓库:不存在则 init,并经 GitWriter 建初始提交让 HEAD 可用。
// gitwriter 只管已存在仓库的提交,仓库本身的初始化用 go-git PlainInit(一次性引导,非提交)。
// cmd/server 启动时调一次。
func EnsureRepo(path string, id Identity) error {
	if path == "" {
		return fmt.Errorf("内容仓库路径为空")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("建内容仓库目录: %w", err)
	}
	w, err := gitwriter.New(path)
	if err != nil {
		return fmt.Errorf("定位内容仓库: %w", err)
	}
	ctx := context.Background()
	head, err := w.Head(ctx)
	if err != nil {
		if !errors.Is(err, gitwriter.ErrNotRepository) {
			return fmt.Errorf("检查内容仓库: %w", err)
		}
		if _, err := git.PlainInit(path, false); err != nil {
			return fmt.Errorf("初始化内容仓库: %w", err)
		}
		head = "" // 新仓库:HEAD 未出生
	}
	if head != "" {
		return nil // 已初始化
	}
	// 初始提交也走 GitWriter(gitwriter 不支持空提交,放一个说明文件占位)。
	if _, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: "", // 未出生 HEAD
		Message:      "chore: 初始化 tally 内容仓库",
		Files: []gitwriter.FileUpdate{{
			Path:    "README.md",
			Content: []byte("# tally 内容仓库\n\n项目、需求、文档与笔记等用户内容的 Git 存储(经 internal/gitwriter 单一写边界提交)。\n"),
		}},
		Signature: id.gitSignature(),
	}); err != nil {
		return fmt.Errorf("初始提交: %w", err)
	}
	return nil
}

// Commit 把 content 写到 relPath(仓库内 / 分隔的相对路径)并创建一个提交,返回提交哈希。
// 经 gitwriter 单一写边界:先读当前 HEAD 作为期望基线,再在校验基线的前提下原子落盘提交。
// 若 HEAD 已离开该基线(并发写入抢先提交),返回包住 gitwriter.ErrBaseRevision 的错误
// (可用 errors.Is 识别),绝不静默叠写到未见过的基线上。
func (c *ContentStore) Commit(relPath string, content []byte, message string) (string, error) {
	if c.RepoPath == "" {
		return "", fmt.Errorf("内容仓库未配置")
	}
	w, err := gitwriter.New(c.RepoPath)
	if err != nil {
		return "", fmt.Errorf("定位内容仓库: %w", err)
	}
	ctx := context.Background()
	base, err := w.Head(ctx)
	if err != nil {
		return "", fmt.Errorf("读取内容仓库状态: %w", err)
	}
	res, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: base,
		Message:      message,
		Files:        []gitwriter.FileUpdate{{Path: relPath, Content: content}},
		Signature:    c.Identity.gitSignature(),
	})
	if err != nil {
		return "", fmt.Errorf("提交 %s: %w", relPath, err)
	}
	return res.Commit, nil
}

// Read 读 relPath 的内容;文件不存在返回 ErrNotFound。
func (c *ContentStore) Read(relPath string) (string, error) {
	b, err := os.ReadFile(filepath.Join(c.RepoPath, filepath.FromSlash(relPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(b), nil
}

// List 返回 dir 下所有以 suffix 结尾的文件的仓库相对路径(/ 分隔,排序)。
// dir 不存在时返回空列表(不是错误)。
func (c *ContentStore) List(dir, suffix string) ([]string, error) {
	root := filepath.Join(c.RepoPath, filepath.FromSlash(dir))
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), suffix) {
			return nil
		}
		rel, err := filepath.Rel(c.RepoPath, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// LastCommit 返回 relPath 最近一次提交的短哈希;无提交或出错返回 ""(best-effort,不报错)。
func (c *ContentStore) LastCommit(relPath string) string {
	repo, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return ""
	}
	name := relPath
	it, err := repo.Log(&git.LogOptions{FileName: &name})
	if err != nil {
		return ""
	}
	commit, err := it.Next()
	if err != nil {
		return ""
	}
	h := commit.Hash.String()
	if len(h) > 7 {
		h = h[:7]
	}
	return h
}

/* ---------- 路径校验(防越界写入) ---------- */

// validSlug 校验标识:小写字母 / 数字 / 连字符,1–64 字符。
// 拒绝 ".."、路径分隔符等一切可能导致越界的输入。
func validSlug(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
		default:
			return false
		}
	}
	return true
}

// validFileName 校验需求文件名:字母 / 数字 / - _ . ,以 .md 结尾,1–128 字符,不含分隔符。
func validFileName(s string) bool {
	if s == "" || len(s) > 128 || !strings.HasSuffix(s, ".md") {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
