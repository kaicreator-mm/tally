package outcome

// outcome 事实持久化适配器(P3-05;migrations/0023 outcome_event,
// PRD §23,决策锁 14,INV-FACT-01):
//
//   - AC1:kind 是封闭枚举,写入前经 Validate 拒绝未知 kind(0023 的
//     CHECK 是第二道防线);读回时未知 kind 视为行破坏,报错而不是静默
//     折叠进已知 kind —— 未知 outcome 不是「大概是 accepted」。
//   - AC2:事件在写入时一次落成 task/change 绑定与 timestamp,之后只读;
//     绑定可得时必填(至少其一),坐标以 NULL 表示缺失(领域侧空串)。
//   - AC3:本适配器不提供任何 Update/Delete API(事实面只追加,0023 的
//     语句级触发器是第二道防线;行写入后只读)。
//   - AC5:本包不落、不缓存任何「当前 outcome」状态:当前结论由调用方
//     对事件流读侧投影,不存在可变权威。
//
// 错误语义:写入前校验失败原样返回;未命中包装 pgx.ErrNoRows;行重建
// (kind 越出 CHECK 取值等)报错而不是跳过 —— outcome 历史丢行比报错
// 更危险(§23:事后无法补采)。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/store"
)

// Store 是 outcome 事件的持久化适配器。只在构造时绑定查询集,本身无状态;
// 用 *store.Queries 构造(事务内传 WithTx 的查询集)。
type Store struct {
	q *store.Queries
}

// New 用 sqlc 生成的查询集构造适配器。
func New(q *store.Queries) *Store {
	return &Store{q: q}
}

// Record 追加一条 outcome 事实,返回新行 id。事件、绑定坐标与写入时间一次
// 落成;未知 kind / 零归属 / 缺时间在写入前被 Validate 拒绝,不触库(AC1/AC2)。
func (s *Store) Record(ctx context.Context, e Event) (int64, error) {
	if err := e.Validate(); err != nil {
		return 0, err
	}
	id, err := s.q.InsertOutcomeEvent(ctx, store.InsertOutcomeEventParams{
		TaskID:      textPtr(e.TaskID),
		ChangesetID: textPtr(e.ChangesetID),
		EventKind:   string(e.Kind),
		RecordedAt:  pgtype.Timestamptz{Time: e.RecordedAt.UTC(), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("outcome: 写入 outcome event: %w", err)
	}
	return id, nil
}

// RecordOnce atomically appends an outcome occurrence if it has not already
// been captured. The uniqueness decision is made by PostgreSQL (migration
// 0025), so concurrent retries cannot both win. inserted=false is a successful
// replay, not an error.
func (s *Store) RecordOnce(ctx context.Context, e Event) (inserted bool, err error) {
	if err := e.Validate(); err != nil {
		return false, err
	}
	_, err = s.q.InsertOutcomeEventOnce(ctx, store.InsertOutcomeEventParams{
		TaskID:      textPtr(e.TaskID),
		ChangesetID: textPtr(e.ChangesetID),
		EventKind:   string(e.Kind),
		RecordedAt:  pgtype.Timestamptz{Time: e.RecordedAt.UTC(), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("outcome: 幂等写入 outcome event: %w", err)
	}
	return true, nil
}

// ForTask 返回某任务全部 outcome 事件(旧→新,recorded_at 同刻按 id 定序):
// accepted→regression_found→… 完整历史可重放,「当前 outcome」由调用方对
// 该事件流做读侧投影(AC5:本包不产出可变状态)。
func (s *Store) ForTask(ctx context.Context, taskID string) ([]Event, error) {
	rows, err := s.q.ListOutcomeEventsByTask(ctx, textPtr(taskID))
	if err != nil {
		return nil, fmt.Errorf("outcome: 查询任务 %s 的 outcome events: %w", taskID, err)
	}
	return eventsFromRows(rows)
}

// ForChangeset 返回某候选 ChangeSet 关联的全部 outcome 事件(旧→新,同刻
// 按 id 定序):候选维度的 outcome 审计(是否被接受、后来是否被回退/发现
// 回归)。
func (s *Store) ForChangeset(ctx context.Context, changesetID string) ([]Event, error) {
	rows, err := s.q.ListOutcomeEventsByChangeset(ctx, textPtr(changesetID))
	if err != nil {
		return nil, fmt.Errorf("outcome: 查询 changeset %s 的 outcome events: %w", changesetID, err)
	}
	return eventsFromRows(rows)
}

func eventsFromRows(rows []store.OutcomeEvent) ([]Event, error) {
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		e, err := eventFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func eventFromRow(row store.OutcomeEvent) (Event, error) {
	e := Event{
		TaskID:      textVal(row.TaskID),
		ChangesetID: textVal(row.ChangesetID),
		Kind:        Kind(row.EventKind),
		RecordedAt:  tsVal(row.RecordedAt),
	}
	// 重建即验证:kind 越出 0023 CHECK 取值的行(行破坏)必须报错,
	// 不被当真、更不被静默折叠(fail-safe,AC1 的读侧镜像)。
	if err := e.Validate(); err != nil {
		return Event{}, fmt.Errorf("outcome: 重建 outcome_event 行 %d: %w", row.ID, err)
	}
	return e, nil
}

// ---- pgtype 换算(本包私有,与 measurement/runstore 的辅助同义) ----

func textPtr(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func textVal(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func tsVal(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}
