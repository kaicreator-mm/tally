package project

// v1.3 派生索引重建(PRD §33.3 Derived Index / INV-STATE-01)。
//
// 把 Git 装载快照(internal/artifacts 的装载产物,合同已过各自领域的校验/冻结)
// 重建进 task_index / verification_index。本文件只认已投影成索引行的输入,不
// import 装载器:派生索引不关心内容从哪个仓库来,只认 (project, task,
// revision/hash, path, source_commit);装载快照 → 索引输入的适配由组合方
// (wiring/集成测试)完成。
//
// 重建遵循 migrations/0018 与 queries 固定的三步合同(一个事务内):
//  1. Clear*IndexCurrent:逐合同退位现任 current(否则 step 2 撞 partial unique index);
//  2. Rebuild*Index:批量 upsert 快照的全部修订行;
//  3. DeleteStale*:删除快照之外的旧行。
//
// 重建后索引内容 = 快照:幂等(同一快照反复重建结果一致),可整体重建
// (清空后按同一 Git 快照重derive即得回同一内容)。因此索引面不持有任何
// 权威状态(AC5):Git 是唯一来源(INV-ART-01),DB 只存可丢弃的投影。
//
// 项目身份:写入前先查 project_index 注册表(§33.3 ghost-project guard,
// ProjectIndexExists)。注册表之外的项目一律拒收,不产生任何索引行 ——
// 索引行的 project_id 只能是 Git 注册表声明过、注册表索引也收录过的身份。

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"tally/internal/store"
)

// V13TaskRevision 是一份要入库的 TaskContract 当前投影行。
type V13TaskRevision struct {
	TaskID       string // 合同 id(如 TASK-023)
	RevisionHash string // 内容修订哈希(taskcontract.RevisionHash,sha256:<hex>)
	Path         string // Git 内制品路径(.tally/tasks/TASK-*.yaml)
}

// V13VerificationRevision 是一份要入库的 VerificationContract 冻结修订行。
type V13VerificationRevision struct {
	TaskID     string
	Revision   int32  // 冻结契约声明的整数 revision(>= 1)
	FrozenHash string // 冻结内容哈希(verification.Frozen.Hash,sha256:<hex>)
	Path       string // Git 内契约路径
}

// V13Snapshot 是一次待入库的 Git 装载快照的索引投影。
//
// 语义:每个合同至多一行(该提交快照的当前投影),is_current 恒为 true;
// 重建后索引内容 = 本快照,快照之外的旧行被 DeleteStale 收走。
type V13Snapshot struct {
	ProjectID     string
	SourceCommit  string // 装载依据的 Git 提交(完整 sha)
	Tasks         []V13TaskRevision
	Verifications []V13VerificationRevision
}

// V13IndexReport 是一次重建的产出统计。
type V13IndexReport struct {
	TasksIndexed         int // 本次快照入库的 TaskContract 当前行数
	VerificationsIndexed int // 本次快照入库的 VerificationContract 冻结修订行数
}

// RebuildV13Indexes 把 v1.3 Git 装载快照重建进 task_index / verification_index。
// 全程一个事务:身份 guard、退位、批量 upsert、清陈旧要么全部生效,要么回滚
// 不留半截索引。不碰事实面:重建只写两张索引表(INV-5 的反面 —— 派生索引
// 必须可整体重写,0018 刻意不挂 append-only 触发器)。
func RebuildV13Indexes(ctx context.Context, st *store.Store, snap V13Snapshot) (V13IndexReport, error) {
	if err := validateV13Snapshot(snap); err != nil {
		return V13IndexReport{}, err
	}
	rep := V13IndexReport{
		TasksIndexed:         len(snap.Tasks),
		VerificationsIndexed: len(snap.Verifications),
	}
	tasks := sortedV13Tasks(snap.Tasks)
	verifs := sortedV13Verifications(snap.Verifications)

	err := st.InTx(ctx, func(q *store.Queries) error {
		// §33.3 ghost-project guard:注册表(project_index)之外的项目,
		// 派生索引拒收 —— 索引行不得为注册表之外的项目身份供料。
		known, err := q.ProjectIndexExists(ctx, snap.ProjectID)
		if err != nil {
			return fmt.Errorf("查项目注册表: %w", err)
		}
		if !known {
			return fmt.Errorf("project: 项目 %q 不在 project_index 注册表,拒绝派生 v1.3 索引(§33.3 ghost-project guard)", snap.ProjectID)
		}
		if err := rebuildV13TaskRows(ctx, q, snap.ProjectID, snap.SourceCommit, tasks); err != nil {
			return err
		}
		return rebuildV13VerificationRows(ctx, q, snap.ProjectID, snap.SourceCommit, verifs)
	})
	if err != nil {
		return V13IndexReport{}, fmt.Errorf("重建 v1.3 派生索引: %w", err)
	}
	return rep, nil
}

// rebuildV13TaskRows 执行 task_index 的三步重建(Clear → Rebuild → DeleteStale)。
// tasks 必须已排序、无重复 task_id(调用方经 validateV13Snapshot/sorted 保证)。
func rebuildV13TaskRows(ctx context.Context, q *store.Queries, projectID, sourceCommit string, tasks []V13TaskRevision) error {
	for _, t := range tasks {
		// step 1:退位现任 current;快照里每个合同的行都带 is_current=true,
		// 不先退位,step 2 会撞 0018 的 partial unique index。
		if err := q.ClearTaskIndexCurrent(ctx, store.ClearTaskIndexCurrentParams{
			ProjectID: projectID,
			TaskID:    t.TaskID,
		}); err != nil {
			return fmt.Errorf("退位 task_index current(%s): %w", t.TaskID, err)
		}
	}

	// step 2:批量 upsert 快照全部行(unnest 数组按位置一一对应)。
	projectIDs := make([]string, len(tasks))
	taskIDs := make([]string, len(tasks))
	revisions := make([]string, len(tasks))
	paths := make([]string, len(tasks))
	current := make([]bool, len(tasks))
	commits := make([]string, len(tasks))
	for i, t := range tasks {
		projectIDs[i] = projectID
		taskIDs[i] = t.TaskID
		revisions[i] = t.RevisionHash
		paths[i] = t.Path
		current[i] = true
		commits[i] = sourceCommit
	}
	if err := q.RebuildTaskIndex(ctx, store.RebuildTaskIndexParams{
		Column1: projectIDs,
		Column2: taskIDs,
		Column3: revisions,
		Column4: paths,
		Column5: current,
		Column6: commits,
	}); err != nil {
		return fmt.Errorf("批量重建 task_index: %w", err)
	}

	// step 3:删除快照之外旧行(合同消失/换修订后,索引内容收窄到快照)。
	if err := q.DeleteStaleTaskIndex(ctx, store.DeleteStaleTaskIndexParams{
		ProjectID: projectID,
		Column2:   taskIDs,
		Column3:   revisions,
	}); err != nil {
		return fmt.Errorf("清理 task_index 陈旧行: %w", err)
	}
	return nil
}

// rebuildV13VerificationRows 执行 verification_index 的三步重建,与
// rebuildV13TaskRows 完全对称(revision 是整数,hash 是冻结内容哈希)。
func rebuildV13VerificationRows(ctx context.Context, q *store.Queries, projectID, sourceCommit string, verifs []V13VerificationRevision) error {
	for _, v := range verifs {
		if err := q.ClearVerificationIndexCurrent(ctx, store.ClearVerificationIndexCurrentParams{
			ProjectID: projectID,
			TaskID:    v.TaskID,
		}); err != nil {
			return fmt.Errorf("退位 verification_index current(%s): %w", v.TaskID, err)
		}
	}

	projectIDs := make([]string, len(verifs))
	taskIDs := make([]string, len(verifs))
	revisions := make([]int32, len(verifs))
	hashes := make([]string, len(verifs))
	paths := make([]string, len(verifs))
	current := make([]bool, len(verifs))
	commits := make([]string, len(verifs))
	for i, v := range verifs {
		projectIDs[i] = projectID
		taskIDs[i] = v.TaskID
		revisions[i] = v.Revision
		hashes[i] = v.FrozenHash
		paths[i] = v.Path
		current[i] = true
		commits[i] = sourceCommit
	}
	if err := q.RebuildVerificationIndex(ctx, store.RebuildVerificationIndexParams{
		Column1: projectIDs,
		Column2: taskIDs,
		Column3: revisions,
		Column4: hashes,
		Column5: paths,
		Column6: current,
		Column7: commits,
	}); err != nil {
		return fmt.Errorf("批量重建 verification_index: %w", err)
	}

	if err := q.DeleteStaleVerificationIndex(ctx, store.DeleteStaleVerificationIndexParams{
		ProjectID: projectID,
		Column2:   taskIDs,
		Column3:   revisions,
		Column4:   hashes,
	}); err != nil {
		return fmt.Errorf("清理 verification_index 陈旧行: %w", err)
	}
	return nil
}

// validateV13Snapshot 校验快照的硬约束:身份与来源可追溯、行字段完整、同一
// 合同在快照内至多一行(当前投影必须唯一)。装载器已按同一规则裁决过一次,
// 这里再拦一道:派生索引的输入合同不依赖调用方自觉。
func validateV13Snapshot(snap V13Snapshot) error {
	if strings.TrimSpace(snap.ProjectID) == "" {
		return fmt.Errorf("project: 快照缺少 project_id")
	}
	if strings.ContainsAny(snap.ProjectID, " \t\r\n") {
		return fmt.Errorf("project: project_id %q 含空白字符", snap.ProjectID)
	}
	if strings.TrimSpace(snap.SourceCommit) == "" {
		return fmt.Errorf("project: 快照缺少 source_commit(派生必须可追溯到 Git 提交,INV-ART-01)")
	}
	taskPaths := map[string]string{}
	for _, t := range snap.Tasks {
		if t.TaskID == "" || t.RevisionHash == "" || t.Path == "" {
			return fmt.Errorf("project: task_index 行字段不完整: %+v", t)
		}
		if prev, dup := taskPaths[t.TaskID]; dup {
			return fmt.Errorf("project: 快照里任务 %s 有两行当前投影(%s 与 %s)", t.TaskID, prev, t.Path)
		}
		taskPaths[t.TaskID] = t.Path
	}
	verifPaths := map[string]string{}
	for _, v := range snap.Verifications {
		if v.TaskID == "" || v.FrozenHash == "" || v.Path == "" {
			return fmt.Errorf("project: verification_index 行字段不完整: %+v", v)
		}
		if v.Revision < 1 {
			return fmt.Errorf("project: 验证契约 %s 的 revision 必须 >= 1(当前 %d)", v.TaskID, v.Revision)
		}
		if prev, dup := verifPaths[v.TaskID]; dup {
			return fmt.Errorf("project: 快照里验证契约 %s 有两行当前投影(%s 与 %s)", v.TaskID, prev, v.Path)
		}
		verifPaths[v.TaskID] = v.Path
	}
	return nil
}

// sortedV13Tasks 返回按 (task_id, path) 排序的拷贝:重建写入顺序确定,
// 同一快照反复重建走完全相同的语句序列。
func sortedV13Tasks(tasks []V13TaskRevision) []V13TaskRevision {
	out := append([]V13TaskRevision(nil), tasks...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].TaskID != out[j].TaskID {
			return out[i].TaskID < out[j].TaskID
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// sortedV13Verifications 返回按 (task_id, revision, hash) 排序的拷贝。
func sortedV13Verifications(verifs []V13VerificationRevision) []V13VerificationRevision {
	out := append([]V13VerificationRevision(nil), verifs...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].TaskID != out[j].TaskID {
			return out[i].TaskID < out[j].TaskID
		}
		if out[i].Revision != out[j].Revision {
			return out[i].Revision < out[j].Revision
		}
		return out[i].FrozenHash < out[j].FrozenHash
	})
	return out
}
