package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tally/internal/core"
	"tally/internal/store"
)

// 两个视图都只读,全部从机器记录算出(PRD 第六部分「和前面几部分怎么连」)。
// 视图 A:某条款的实现累计失败。视图 B:某指纹的影响面。

var (
	// ErrClauseNotFound:项目下没有该条款的任何版本(视图 A 无从算起)。
	ErrClauseNotFound = errors.New("条款不存在(无签署版本)")
	// ErrFingerprintNotFound:指纹索引里没有该精确指纹(视图 B 无从算起)。
	ErrFingerprintNotFound = errors.New("指纹不存在")
)

// FailureComponent 是视图 A 失败构成的一项:一个指纹在本条款实现链上出现几次、当前处置分类。
type FailureComponent struct {
	Fingerprint    core.Fingerprint
	Count          int64
	Classification string // fingerprint_stat.status;尚未归类时为 unresolved
}

// ClauseImplReport 是视图 A:某条款的实现累计失败。
type ClauseImplReport struct {
	ProjectID core.ProjectID
	ClauseID  string

	ImplCommits     int64 // 实现提交数(挂到本条款任一版本的 commit 去重数)
	BuildFailures   int64 // 构建失败数
	DeployFailures  int64 // 部署失败数
	SameFingerprint int64 // 其中同一指纹的最高复发次数(失败越集中,越可能是需求没写清)
	ChangeCount     int64 // 条款变更次数(签署过的版本数)
	FirstSignedAt   time.Time
	AgeDays         int // 条款年龄:首次签署至今的天数

	Failures []FailureComponent // 失败构成,按指纹列出并标注处置分类
}

// ClauseImpl 算视图 A。条款按逻辑编号(project_id + clause_id)聚合,跨其全部签署版本 ——
// 因为「条款年龄与变更次数」本就是跨版本的概念;挂钩记的是各版本的 content_hash(INV-4),
// 这里把它们都算进本条款的实现。
func ClauseImpl(ctx context.Context, q *store.Queries, projectID core.ProjectID, clauseID string) (ClauseImplReport, error) {
	row, err := q.ViewClauseImpl(ctx, store.ViewClauseImplParams{
		ProjectID: string(projectID), ClauseID: clauseID,
	})
	if err != nil {
		return ClauseImplReport{}, fmt.Errorf("算视图 A(%s): %w", clauseID, err)
	}
	if row.ChangeCount == 0 {
		return ClauseImplReport{}, ErrClauseNotFound
	}

	report := ClauseImplReport{
		ProjectID:       projectID,
		ClauseID:        clauseID,
		ImplCommits:     row.ImplCommits,
		BuildFailures:   row.BuildFailures,
		DeployFailures:  row.DeployFailures,
		SameFingerprint: row.TopFpRecurrence,
		ChangeCount:     row.ChangeCount,
	}
	if row.FirstSignedAt.Valid {
		report.FirstSignedAt = row.FirstSignedAt.Time
		report.AgeDays = int(time.Since(row.FirstSignedAt.Time).Hours() / 24)
	}

	fails, err := q.ViewClauseImplFailures(ctx, store.ViewClauseImplFailuresParams{
		ProjectID: string(projectID), ClauseID: clauseID,
	})
	if err != nil {
		return ClauseImplReport{}, fmt.Errorf("算视图 A 失败构成(%s): %w", clauseID, err)
	}
	for _, f := range fails {
		report.Failures = append(report.Failures, FailureComponent{
			Fingerprint:    core.Fingerprint(f.FpExact),
			Count:          f.Occurrence,
			Classification: f.Classification,
		})
	}
	return report, nil
}

// RelatedClause 是视图 B 里指纹关联到的一条条款(经失败→提交→挂钩推得)。
type RelatedClause struct {
	ClauseID    string
	ContentHash core.ClauseHash
}

// FingerprintImpact 是视图 B:某指纹的影响面。
type FingerprintImpact struct {
	Fingerprint core.Fingerprint

	Occurrences          int64
	AffectedProjects     []string
	AffectedProjectCount int
	Status               string // 处置分类

	HasFix        bool
	FixRef        string            // 指向 git 里已验证修复的引用
	VerifiedAt    time.Time         // 修复验证时刻(有效性边界)
	VerifiedCombo core.VersionCombo // 修复验证时的版本组合
	Drifted       bool              // 修复验证后又复发(末次出现晚于验证时刻)→ 修复可能过期

	RelatedClauses []RelatedClause // 关联条款;无需求层时为空(LEFT JOIN,其余字段照常)
}

// FingerprintImpactView 算视图 B。
//
// 关联条款来自 LEFT JOIN:指纹的失败经 build/deploy 回到 commit,再经 commit_clause 到条款。
// 没有需求层(推不出条款)时 RelatedClauses 为空,但出现次数/影响项目/修复等照常返回 ——
// 这正是 PRD 说的「横跨 N 个项目不需要第六部分就能答,加上第六部分才知道影响哪几条需求」。
func FingerprintImpactView(ctx context.Context, q *store.Queries, fp core.Fingerprint) (FingerprintImpact, error) {
	rows, err := q.ViewFingerprintImpact(ctx, string(fp))
	if err != nil {
		return FingerprintImpact{}, fmt.Errorf("算视图 B(%s): %w", fp, err)
	}
	if len(rows) == 0 {
		return FingerprintImpact{}, ErrFingerprintNotFound
	}

	head := rows[0] // 指纹级字段每行相同,取首行
	impact := FingerprintImpact{
		Fingerprint:          core.Fingerprint(head.FpExact),
		Occurrences:          head.OccurrenceCount,
		AffectedProjects:     head.AffectedProjects,
		AffectedProjectCount: len(head.AffectedProjects),
		Status:               head.Status,
		Drifted:              head.Drifted.Bool,
	}
	if head.FixRef.Valid {
		impact.HasFix = true
		impact.FixRef = head.FixRef.String
	}
	if head.FixVerifiedAt.Valid {
		impact.VerifiedAt = head.FixVerifiedAt.Time
	}
	if len(head.FixVerifiedCombo) > 0 {
		if err := json.Unmarshal(head.FixVerifiedCombo, &impact.VerifiedCombo); err != nil {
			return FingerprintImpact{}, fmt.Errorf("解析修复验证组合(%s): %w", fp, err)
		}
	}

	// 关联条款:多行里收集非空的条款(LEFT JOIN 无匹配时为 NULL,跳过),去重保序。
	seen := map[string]bool{}
	for _, r := range rows {
		if !r.RelatedClauseID.Valid || seen[r.RelatedClauseID.String] {
			continue
		}
		seen[r.RelatedClauseID.String] = true
		impact.RelatedClauses = append(impact.RelatedClauses, RelatedClause{
			ClauseID:    r.RelatedClauseID.String,
			ContentHash: core.ClauseHash(r.RelatedClauseHash.String),
		})
	}
	return impact, nil
}
