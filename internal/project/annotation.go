package project

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/store"
)

// 测试注解 `@covers REQ-12.2` 把测试挂到条款上,用于算条款覆盖率。
// 与 trailer 一样,挂钩记 content_hash 不记逻辑编号(INV-4)。

// reCovers 匹配 @covers 注解及其后同一行内的条款引用,如 `// @covers REQ-12.2`。
// \b 保证 @coversFoo 不误匹配。
var reCovers = regexp.MustCompile(`@covers\b([^\n]*)`)

// CoversFromSource 从测试源码里抽出 @covers 注解引用的条款编号。
// 支持一行多个(逗号/空白分隔)与文件中多处注解,去重保序。纯函数,不碰 git。
func CoversFromSource(src string) []string {
	var refs []string
	seen := map[string]bool{}
	for _, m := range reCovers.FindAllStringSubmatch(src, -1) {
		for _, ref := range extractClauseRefs(m[1]) {
			if !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// BindTest 把一个测试(按名字标识)引用的条款,按时刻 at 的当前版本绑成 content_hash,
// 写入 test_clause(INV-4)。返回绑定数。当时无签署版本的引用被跳过。
func BindTest(ctx context.Context, q *store.Queries, projectID core.ProjectID, testName string, refs []string, at time.Time) (int, error) {
	bound := 0
	for _, ref := range refs {
		hash, err := q.CurrentClauseVersionHashAt(ctx, store.CurrentClauseVersionHashAtParams{
			ProjectID: string(projectID),
			ClauseID:  ref,
			SignedAt:  pgtype.Timestamptz{Time: at, Valid: true},
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return bound, fmt.Errorf("查条款 %s 当前版本(测试 %s): %w", ref, testName, err)
		}
		err = q.InsertTestClause(ctx, store.InsertTestClauseParams{
			ProjectID:         string(projectID),
			TestName:          testName,
			ClauseContentHash: hash,
		})
		if err != nil {
			return bound, fmt.Errorf("写 test_clause(%s→%s): %w", testName, ref, err)
		}
		bound++
	}
	return bound, nil
}

// maxScanBytes 是注解扫描跳过的文件大小上限:超过的多半是二进制/生成物,不值得读。
const maxScanBytes = 1 << 20 // 1 MiB

// ScanAnnotations 用 go-git 读仓库 HEAD 树的全部文件,扫 @covers 注解,按 HEAD 提交时刻
// 绑定当前条款版本写入 test_clause。测试名用仓库内相对路径。
//
// 覆盖反映「当前测试套件对当前条款」的覆盖,所以统一按 HEAD 时刻绑定当前版本 ——
// 与 trailer 按各提交历史时刻绑定不同(那是要还原历史挂钩)。返回绑定总数。
func ScanAnnotations(ctx context.Context, q *store.Queries, projectID core.ProjectID, repoPath string) (int, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0, fmt.Errorf("打开仓库 %s: %w", repoPath, err)
	}
	head, err := repo.Head()
	if err != nil {
		return 0, fmt.Errorf("读 HEAD: %w", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return 0, fmt.Errorf("读 HEAD 提交: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return 0, fmt.Errorf("读 HEAD 树: %w", err)
	}
	when := commit.Author.When
	total := 0
	err = tree.Files().ForEach(func(f *object.File) error {
		if f.Size > maxScanBytes {
			return nil // 跳过大文件
		}
		content, readErr := f.Contents()
		if readErr != nil {
			// 读不了的(如二进制)跳过,不让一个坏文件挡掉整库扫描 —— 故意的,非漏报错误。
			return nil //nolint:nilerr // 见上行注释:跳过不可读文件是预期行为
		}
		refs := CoversFromSource(content)
		if len(refs) == 0 {
			return nil
		}
		n, bindErr := BindTest(ctx, q, projectID, f.Name, refs, when)
		if bindErr != nil {
			return bindErr
		}
		total += n
		return nil
	})
	if err != nil {
		return total, fmt.Errorf("扫描测试注解: %w", err)
	}
	return total, nil
}

// Coverage 是条款覆盖率快照:签署的当前条款里有多少被测试 @covers 引用。
type Coverage struct {
	Total     int      // 当前条款数(is_current)
	Covered   int      // 有测试引用其当前版本 content_hash 的条款数
	Uncovered []string // 未被任何测试引用的条款 id
}

// Rate 返回覆盖率(0..1);没有条款时为 0(不当除零错误,空项目就是 0)。
func (c Coverage) Rate() float64 {
	if c.Total == 0 {
		return 0
	}
	return float64(c.Covered) / float64(c.Total)
}

// ClauseCoverage 算某项目的条款覆盖率:当前条款(is_current)里,test_clause 引用了其
// content_hash 的算覆盖。test_clause 无项目列,但匹配走的是本项目条款的 content_hash,
// 天然不跨项目串。没有需求层(无当前条款)时 Total=0,Rate=0。
func ClauseCoverage(ctx context.Context, q *store.Queries, projectID core.ProjectID) (Coverage, error) {
	currents, err := q.ListCurrentClauseVersions(ctx, string(projectID))
	if err != nil {
		return Coverage{}, fmt.Errorf("列当前条款: %w", err)
	}
	tests, err := q.ListTestClauses(ctx)
	if err != nil {
		return Coverage{}, fmt.Errorf("列测试挂钩: %w", err)
	}
	coveredHash := make(map[string]bool, len(tests))
	for _, t := range tests {
		coveredHash[t.ClauseContentHash] = true
	}
	var cov Coverage
	cov.Total = len(currents)
	for _, c := range currents {
		if coveredHash[c.ContentHash] {
			cov.Covered++
		} else {
			cov.Uncovered = append(cov.Uncovered, c.ClauseID)
		}
	}
	return cov, nil
}
