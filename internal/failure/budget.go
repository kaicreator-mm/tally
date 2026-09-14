package failure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/store"
)

// maxAttempts 是诊断预算:提方案 → 验证不过 → 再提,这个循环最多三轮。
// 超了归档「暂时没有自动修复」,同指纹再命中直接返回结论,不再调 LLM
// (PRD 第四部分规则 ⑤:不设上限,这是整个工具里唯一会闷声烧钱的地方)。
const maxAttempts = 3

// ErrNoStat:指纹索引行不存在(该指纹的第一次失败)。
var ErrNoStat = errors.New("指纹索引行不存在")

// Stat 是指纹索引行里 failure 关心的部分。做成值类型而不直接用 store 的行类型,
// 预算逻辑才能在单测里用内存 fake 离线跑;真实实现是 statStore(包装 store.Queries)。
type Stat struct {
	Status           string
	DiagnoseAttempts int
	AffectedProjects []string
	FixRef           string
}

// BudgetDecision 是预算闸门的判定(确定性,不触 LLM)。
type BudgetDecision struct {
	Proceed   bool         // 允许再来一轮诊断
	Archived  bool         // 三轮用尽:暂无自动修复
	Concluded *OutcomeKind // 终态出口:直接返回该结论
}

// ConsultBudget 从索引现状判断「还要不要继续诊断」:
// 终态出口 → 直接返回该结论;三轮用尽 → 归档;否则放行一轮。
// 前两种情况不产生任何 LLM 调用。
func ConsultBudget(found bool, stat Stat) BudgetDecision {
	if !found {
		return BudgetDecision{Proceed: true}
	}
	if IsTerminal(stat.Status) {
		k := OutcomeKind(stat.Status)
		return BudgetDecision{Concluded: &k}
	}
	if stat.DiagnoseAttempts >= maxAttempts {
		return BudgetDecision{Archived: true}
	}
	return BudgetDecision{Proceed: true}
}

// StatStore 是 failure 对指纹索引的期望(使用方定义)。真实实现 statStore 包装
// store.Queries;单测注入内存 fake。HasRecentDepChange 提供「上游问题」归类的
// 确定性信号:失败与依赖升级在时间线上对齐。
type StatStore interface {
	// GetStat 取索引行;行不存在返回 ErrNoStat。
	GetStat(ctx context.Context, fp core.Fingerprint) (Stat, error)
	BumpAttempts(ctx context.Context, fp core.Fingerprint) error
	SetClassification(ctx context.Context, fp core.Fingerprint, kind OutcomeKind) error
	SetFix(ctx context.Context, fp core.Fingerprint, fixRef string, verifiedAt time.Time, combo core.VersionCombo) error
	HasRecentDepChange(ctx context.Context, projectID string, since time.Time) (bool, error)
}

// NewStatStore 把 store.Queries 包成 StatStore。
func NewStatStore(q *store.Queries) StatStore { return &statStore{q: q} }

type statStore struct{ q *store.Queries }

func (a *statStore) GetStat(ctx context.Context, fp core.Fingerprint) (Stat, error) {
	row, err := a.q.GetFingerprintStat(ctx, string(fp))
	if errors.Is(err, pgx.ErrNoRows) {
		return Stat{}, ErrNoStat
	}
	if err != nil {
		return Stat{}, fmt.Errorf("读指纹索引: %w", err)
	}
	return Stat{
		Status:           row.Status,
		DiagnoseAttempts: int(row.DiagnoseAttempts),
		AffectedProjects: row.AffectedProjects,
		FixRef:           row.FixRef.String,
	}, nil
}

func (a *statStore) BumpAttempts(ctx context.Context, fp core.Fingerprint) error {
	if err := a.q.BumpDiagnoseAttempts(ctx, string(fp)); err != nil {
		return fmt.Errorf("累加诊断次数: %w", err)
	}
	return nil
}

func (a *statStore) SetClassification(ctx context.Context, fp core.Fingerprint, kind OutcomeKind) error {
	err := a.q.UpdateFingerprintStatClassification(ctx, store.UpdateFingerprintStatClassificationParams{
		FpExact: string(fp),
		Status:  string(kind),
	})
	if err != nil {
		return fmt.Errorf("写归类 %s: %w", kind, err)
	}
	return nil
}

func (a *statStore) SetFix(ctx context.Context, fp core.Fingerprint, fixRef string, verifiedAt time.Time, combo core.VersionCombo) error {
	raw, err := json.Marshal(combo)
	if err != nil {
		return fmt.Errorf("序列化验证组合: %w", err)
	}
	err = a.q.UpdateFingerprintStatFix(ctx, store.UpdateFingerprintStatFixParams{
		FpExact:          string(fp),
		FixRef:           pgtype.Text{String: fixRef, Valid: fixRef != ""},
		FixVerifiedAt:    pgtype.Timestamptz{Time: verifiedAt, Valid: !verifiedAt.IsZero()},
		FixVerifiedCombo: raw,
	})
	if err != nil {
		return fmt.Errorf("写修复索引: %w", err)
	}
	return nil
}

func (a *statStore) HasRecentDepChange(ctx context.Context, projectID string, since time.Time) (bool, error) {
	n, err := a.q.CountDependencyChangesSince(ctx, store.CountDependencyChangesSinceParams{
		ProjectID: projectID,
		ChangedAt: pgtype.Timestamptz{Time: since, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("查依赖变更: %w", err)
	}
	return n > 0, nil
}

// RecordEvent 把失败写进事实面(failure_event,只追加 INV-5)并累加索引面
// (fingerprint_stat),同一事务。索引是查库/预算/归类的地基,必须与事件同生。
// 事实面只由系统在执行动作时 INSERT,界面不碰(界面的写边界)。
func RecordEvent(ctx context.Context, s *store.Store, ev Event) (int64, error) {
	sources := 0
	for _, id := range []int64{ev.BuildRunID, ev.DeployRunID, ev.EnvironmentRunID} {
		if id != 0 {
			sources++
		}
	}
	if sources != 1 {
		return 0, fmt.Errorf("记录失败事件:三个来源外键必须恰好一个非空, got %d", sources)
	}
	combo, err := json.Marshal(ev.Combo)
	if err != nil {
		return 0, fmt.Errorf("序列化版本组合: %w", err)
	}
	// 复现上下文(§21.6 结构化合同):只落结构化列。repro_command 是 CommandSpec
	// 规范 JSON(internal/command 的 MarshalJSON 产物);Argv 为空(没有可复现命令)
	// 落 JSON null(0016 的「未记录结构化复现」哨兵)。other_target_commands 是
	// [{"target":…,"command":…}] 数组,空/nil 一律记空数组。旧文本三列不再写入
	// (NOT NULL DEFAULT ''),仅作存量行兼容读取来源。随 INSERT 落库,不提供 UPDATE(INV-5)。
	reproCommand := []byte("null")
	if len(ev.Repro.Argv) > 0 {
		reproCommand, err = json.Marshal(ev.Repro)
		if err != nil {
			return 0, fmt.Errorf("序列化复现命令: %w", err)
		}
	}
	targets := []byte("[]") // 空/nil 一律记空数组(列 NOT NULL,也避免 null)
	if len(ev.OtherTargets) > 0 {
		targets, err = json.Marshal(ev.OtherTargets)
		if err != nil {
			return 0, fmt.Errorf("序列化其余构建目标: %w", err)
		}
	}
	var id int64
	err = s.InTx(ctx, func(q *store.Queries) error {
		inserted, err := q.InsertFailureEvent(ctx, store.InsertFailureEventParams{
			ProjectID:           ev.ProjectID,
			Source:              ev.Source,
			BuildRunID:          nullableID(ev.BuildRunID),
			DeployRunID:         nullableID(ev.DeployRunID),
			EnvironmentRunID:    nullableID(ev.EnvironmentRunID),
			RawLogRef:           nullableText(ev.RawLogRef),
			RawLogBytes:         pgtype.Int8{Int64: ev.RawLogBytes, Valid: ev.RawLogBytes > 0},
			Excerpt:             nullableText(ev.Excerpt),
			Normalized:          nullableText(ev.Normalized),
			FpExact:             string(ev.Exact),
			FpFuzzy:             string(ev.Fuzzy),
			PipelineVersion:     ev.PipelineVersion,
			OccurredAt:          pgtype.Timestamptz{Time: ev.OccurredAt, Valid: !ev.OccurredAt.IsZero()},
			VersionCombo:        combo,
			ReproCommand:        reproCommand,
			OtherTargetCommands: targets,
		})
		if err != nil {
			return fmt.Errorf("写 failure_event: %w", err)
		}
		id = inserted
		_, err = q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
			FpExact:   string(ev.Exact),
			FpFuzzy:   string(ev.Fuzzy),
			FirstSeen: pgtype.Timestamptz{Time: ev.OccurredAt, Valid: !ev.OccurredAt.IsZero()},
			Project:   ev.ProjectID,
		})
		if err != nil {
			return fmt.Errorf("累加指纹索引: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func nullableID(id int64) pgtype.Int8   { return pgtype.Int8{Int64: id, Valid: id != 0} }
func nullableText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
