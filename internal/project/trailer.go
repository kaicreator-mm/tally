package project

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/store"
)

// 挂钩来自机器记录(commit trailer 与测试注解),不来自人点(INV-3/INV-7)。
// 绑定一律落「提交当时那一版」的 content_hash,绑定后不再变(INV-4):
// 条款后来升级,旧挂钩仍指向旧版本的哈希,下游自动显示「依据已变更」。

// reTrailerLine 匹配 trailer 段里的 Refs 行,如 `Refs: REQ-12`。键大小写不敏感。
var reTrailerLine = regexp.MustCompile(`(?i)^refs:\s*(.+)$`)

// reClauseRef 是一条可引用的条款编号:REQ-12、REQ-12.2(与 requirement 解析出的条款 id 同形)。
// 只认数字点分,避免把正文里的 "REQ-xxx" 噪声当挂钩。
var reClauseRef = regexp.MustCompile(`^REQ-[0-9]+(?:\.[0-9]+)*$`)

// refSeparators 切开一个 Refs 值里的多个条款引用(逗号/空白/顿号/分号)。
func refSeparators(r rune) bool {
	switch r {
	case ',', ' ', '\t', ';', '、', '，':
		return true
	}
	return false
}

// cleanRefToken 去掉条款引用首尾的标点,让它能匹配 reClauseRef(如 "REQ-12." → "REQ-12")。
func cleanRefToken(tok string) string {
	return strings.Trim(tok, ".,;:、。")
}

// extractClauseRefs 从一个 trailer 值里抽出合法条款引用,去重保序。
func extractClauseRefs(value string) []string {
	var refs []string
	seen := map[string]bool{}
	for _, tok := range strings.FieldsFunc(value, refSeparators) {
		tok = cleanRefToken(tok)
		if reClauseRef.MatchString(tok) && !seen[tok] {
			seen[tok] = true
			refs = append(refs, tok)
		}
	}
	return refs
}

// RefsFromMessage 从提交信息里抽出条款引用(来自尾部的 `Refs:` trailer)。
//
// git trailer 约定在消息最后一段(以空行与正文分隔),所以只看最后一段里的 Refs 行,
// 避免把正文里偶然出现的 "Refs:" 当成挂钩。支持一行多个引用、多行 Refs。纯函数,不碰 git。
func RefsFromMessage(msg string) []string {
	var refs []string
	seen := map[string]bool{}
	for _, line := range strings.Split(lastParagraph(msg), "\n") {
		m := reTrailerLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		for _, ref := range extractClauseRefs(m[1]) {
			if !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

// lastParagraph 取消息的最后一段:按空行(可含空白)切段,返回最后一个非空段。
// 没有空行时整条消息就是最后一段。
func lastParagraph(msg string) string {
	msg = strings.TrimRight(msg, "\n")
	blocks := regexp.MustCompile(`\n[ \t]*\n`).Split(msg, -1)
	for i := len(blocks) - 1; i >= 0; i-- {
		if strings.TrimSpace(blocks[i]) != "" {
			return blocks[i]
		}
	}
	return msg
}

// BindCommit 解析一个提交的 Refs trailer,按提交时刻 at 的当前条款版本把每个引用绑成
// content_hash,写入 commit_clause(INV-4:绑定后不变)。返回成功绑定的条款数。
//
// 指向「当时还没有签署版本」的条款的引用被跳过(绑不了),不算错误 ——
// 这正是 trailer 先于签署、或引用了别处条款时的正常情形。
func BindCommit(ctx context.Context, q *store.Queries, projectID core.ProjectID, sha, msg string, at time.Time) (int, error) {
	bound := 0
	for _, ref := range RefsFromMessage(msg) {
		hash, err := q.CurrentClauseVersionHashAt(ctx, store.CurrentClauseVersionHashAtParams{
			ProjectID: string(projectID),
			ClauseID:  ref,
			SignedAt:  pgtype.Timestamptz{Time: at, Valid: true},
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue // 当时无签署版本,跳过
			}
			return bound, fmt.Errorf("查条款 %s 在提交 %s 时刻的当前版本: %w", ref, sha, err)
		}
		err = q.InsertCommitClause(ctx, store.InsertCommitClauseParams{
			ProjectID:         string(projectID),
			CommitSha:         sha,
			ClauseContentHash: hash,
		})
		if err != nil {
			return bound, fmt.Errorf("写 commit_clause(%s→%s): %w", sha, ref, err)
		}
		bound++
	}
	return bound, nil
}

// IndexTrailers 用 go-git 读仓库全部提交,逐个解析 Refs trailer 并绑定,写入 commit_clause。
// 每个提交按其作者时间(提交真正产生的时刻)绑定当时的当前版本(INV-4)。返回总绑定数。
//
// 重算时由 reindex 先按项目清掉旧挂钩再调本函数,保证挂钩是 git 的纯函数(单向数据流)。
func IndexTrailers(ctx context.Context, q *store.Queries, projectID core.ProjectID, repoPath string) (int, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return 0, fmt.Errorf("打开仓库 %s: %w", repoPath, err)
	}
	iter, err := repo.CommitObjects()
	if err != nil {
		return 0, fmt.Errorf("读提交对象: %w", err)
	}
	total := 0
	err = iter.ForEach(func(c *object.Commit) error {
		n, bindErr := BindCommit(ctx, q, projectID, c.Hash.String(), c.Message, c.Author.When)
		if bindErr != nil {
			return bindErr
		}
		total += n
		return nil
	})
	if err != nil {
		return total, fmt.Errorf("遍历提交挂钩: %w", err)
	}
	return total, nil
}
