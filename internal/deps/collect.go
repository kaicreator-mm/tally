package deps

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/store"
)

// lockfiles 是参与采集的锁文件(D-a1):docker/compose/kamal 没有锁文件,无从 diff。
var lockfiles = []struct {
	Path  string
	Parse func([]byte) (map[string]string, error)
}{
	{"go.mod", ParseGoMod},
	{"package-lock.json", ParsePackageLock},
}

// Options 是一次采集的坐标与区间。
type Options struct {
	ProjectID string
	RepoPath  string
	From      string // 起始 rev(**不含**,上次采到这里);空 = 从仓库开头
	To        string // 结束 rev(含);空 = HEAD
}

// Collect 遍历区间内提交,逐个与父提交比锁文件内容,变更写进 dependency_change。
// 返回本次实际新写入的包名列表(唯一索引去重后)与新行数。
// 新写入的包名是退休检查的触发集(D-c1):重跑同一区间去重后为空,不会重复重验。
//
// 事实面只 INSERT(INV-5);重跑安全(ON CONFLICT DO NOTHING)。
func Collect(ctx context.Context, s *store.Store, opts Options) ([]string, int, error) {
	if opts.ProjectID == "" {
		return nil, 0, errors.New("采集依赖变更:缺少 project_id")
	}
	if opts.RepoPath == "" {
		return nil, 0, errors.New("采集依赖变更:缺少仓库路径")
	}
	repo, err := git.PlainOpen(opts.RepoPath)
	if err != nil {
		return nil, 0, fmt.Errorf("打开仓库 %s: %w", opts.RepoPath, err)
	}

	toHash, err := resolveRev(repo, opts.To, true)
	if err != nil {
		return nil, 0, fmt.Errorf("解析结束 rev: %w", err)
	}
	var fromHash plumbing.Hash
	if opts.From != "" {
		fromHash, err = resolveRev(repo, opts.From, false)
		if err != nil {
			return nil, 0, fmt.Errorf("解析起始 rev: %w", err)
		}
	}

	iter, err := repo.Log(&git.LogOptions{From: toHash})
	if err != nil {
		return nil, 0, fmt.Errorf("遍历提交: %w", err)
	}

	changed := map[string]bool{}
	inserted := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if fromHash != plumbing.ZeroHash && c.Hash == fromHash {
			return errStopRange // From 本身不含(上次已采)
		}
		if c.NumParents() == 0 {
			return nil // 根提交无父可比:全量「新增」是噪声不是升级信号(D-a3)
		}
		n, cerr := collectCommit(ctx, s, repo, opts.ProjectID, c, changed)
		inserted += n
		return cerr
	})
	if errors.Is(err, errStopRange) {
		err = nil
	}
	if err != nil {
		return nil, inserted, fmt.Errorf("采集 %s: %w", opts.ProjectID, err)
	}
	return sortedKeys(changed), inserted, nil
}

var errStopRange = errors.New("到达起始 rev")

// sortedKeys 返回集合的排序键列表(确定序,重跑可 diff)。
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// resolveRev 把 rev 字符串解析成提交哈希;空串按 wantHead 取 HEAD 或报缺失。
func resolveRev(repo *git.Repository, rev string, wantHead bool) (plumbing.Hash, error) {
	if rev == "" {
		if !wantHead {
			return plumbing.ZeroHash, errors.New("缺少 rev")
		}
		head, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, err
		}
		return head.Hash(), nil
	}
	h, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return *h, nil
}

// collectCommit 比对一个提交与其父提交(合并取第一父)的锁文件,变更入库。
// 实际新写入的包名记进 changed(退休检查的触发集)。
func collectCommit(ctx context.Context, s *store.Store, repo *git.Repository, project string, c *object.Commit, changed map[string]bool) (int, error) {
	parent := c.ParentHashes[0]
	inserted := 0
	for _, lf := range lockfiles {
		cur, err := fileAt(repo, c.Hash, lf.Path)
		if err != nil {
			return inserted, fmt.Errorf("读 %s@%s: %w", lf.Path, c.Hash.String()[:8], err)
		}
		prev, err := fileAt(repo, parent, lf.Path)
		if err != nil {
			return inserted, fmt.Errorf("读 %s@%s: %w", lf.Path, parent.String()[:8], err)
		}
		if bytes.Equal(cur, prev) {
			continue
		}
		before, err := lf.Parse(prev)
		if err != nil {
			return inserted, fmt.Errorf("%s@%s: %w", lf.Path, parent.String()[:8], err)
		}
		after, err := lf.Parse(cur)
		if err != nil {
			return inserted, fmt.Errorf("%s@%s: %w", lf.Path, c.Hash.String()[:8], err)
		}
		for _, ch := range Diff(before, after) {
			n, err := s.Queries().InsertDependencyChangeIfAbsent(ctx, store.InsertDependencyChangeIfAbsentParams{
				ProjectID:   project,
				CommitSha:   c.Hash.String(),
				Package:     ch.Package,
				FromVersion: pgtype.Text{String: ch.From, Valid: ch.From != ""},
				ToVersion:   pgtype.Text{String: ch.To, Valid: ch.To != ""},
				// 升级发生时刻 = 提交时刻,不是采集时刻(D-a4)。
				ChangedAt: pgtype.Timestamptz{Time: c.Committer.When.UTC(), Valid: true},
			})
			if err != nil {
				return inserted, fmt.Errorf("写依赖变更 %s: %w", ch.Package, err)
			}
			inserted += int(n) // ON CONFLICT DO NOTHING:已存在的行返回 0,天然去重
			if n > 0 {
				changed[ch.Package] = true // 本次新写入 → 纳入退休触发集
			}
		}
	}
	return inserted, nil
}

// fileAt 读某提交树里的文件内容;文件当时不存在(如首次引入前)按空内容处理。
func fileAt(repo *git.Repository, h plumbing.Hash, path string) ([]byte, error) {
	commit, err := repo.CommitObject(h)
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}
	f, err := tree.File(path)
	if err != nil {
		if errors.Is(err, object.ErrFileNotFound) {
			return nil, nil
		}
		return nil, err
	}
	content, err := f.Contents()
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}
