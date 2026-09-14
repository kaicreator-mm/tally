// audit.go 拥有集成事实的只读审计列举(P5-05): Accepted(验收)与
// Integrated(已集成)是两个可分别观察的工程状态 —— 本文件只提供 Integrated
// 侧的 Git 事实列举(deterministic integration branches,P5-02 落地的
// refs/heads/tally/integration/*),绝不把验收事实与集成事实合并或互相推导。
//
// 边界(改动前必读):
//   - 只读:for-each-ref 枚举 + 逐 ref 读提交对象;无任何写路径(写面归
//     P5-02/P5-03 的具名操作);
//   - 列举范围锁定 refs/heads/tally/integration/ 前缀(治理命名空间,
//     P5-02 确定性命名);其他引用一律不见;
//   - 输出按 ref 名升序(确定性);Ref 名内嵌 ChangeSetID 的 sha256 前缀,
//     调用方可据此与验收事实对账(对账本身是 UI/调用方职责,本层不合并)。
package gitintegration

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"tally/internal/changeset"
	"tally/internal/gitwriter"
)

// IntegrationRef 是一条已落地的集成分支事实。
type IntegrationRef struct {
	// Ref 是完整引用名(refs/heads/tally/integration/<sha256>)。
	Ref string
	// Commit 是该引用指向的提交 sha。
	Commit string
	// Subject 是提交主题行(观测值,原样呈现)。
	Subject string
	// Date 是提交时刻(iso-strict 原文)。
	Date string
}

// ListIntegrationRefs 枚举仓库中 refs/heads/tally/integration/ 下的全部集成
// 分支(按 ref 名升序)。空集合 = 尚无集成事实,不是错误。
func ListIntegrationRefs(ctx context.Context, repoPath string) ([]IntegrationRef, error) {
	out, err := gitwriter.RunGit(ctx, repoPath, "", "for-each-ref",
		"--format=%(refname)%00%(objectname)%00%(subject)%00%(creatordate:iso-strict)",
		"refs/heads/tally/integration/")
	if err != nil {
		return nil, fmt.Errorf("gitintegration: 枚举集成分支: %w", err)
	}
	var refs []IntegrationRef
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("gitintegration: for-each-ref 输出格式异常: %q", line)
		}
		refs = append(refs, IntegrationRef{
			Ref:     parts[0],
			Commit:  parts[1],
			Subject: parts[2],
			Date:    parts[3],
		})
	}
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].Ref < refs[j].Ref })
	return refs, nil
}

// IntegrationBranchRefFor 返回 ChangeSetID 对应的确定性集成分支引用名
// (P5-02 命名规则的导出面,供验收侧与集成侧对账;不执行任何 Git 操作)。
func IntegrationBranchRefFor(id changeset.ChangeSetID) (string, error) {
	return integrationBranchRef(id)
}
