package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
	"tally/internal/store"
)

// build_run.status 的取值。表上没有 CHECK 约束,但录入只产出这两个。
const (
	StatusPassed = "passed"
	StatusFailed = "failed"
)

// BuildInput 在 Input 之上补 build_run 专属字段。
//
// Input.Source 被忽略:构建录入恒为 source=build。三个来源外键里 BuildRunID 由
// IngestBuild 在写完 build_run 后自己填,调用方不用给。
type BuildInput struct {
	Input
	CommitSha      string    // NOT NULL:构建必须能回指 commit(追溯链坐标)
	CiRunURL       string    // CI 运行链接(可空)
	Status         string    // passed / failed:只有 failed 才录 failure_event
	ExitCode       int       // 构建退出码
	StartedAt      time.Time // NOT NULL:缺了直接报错,不落半行
	FinishedAt     time.Time // 可空(零值 → NULL)
	ArtifactDigest string    // 成功时的制品 digest(部署只认这个)
}

// IngestBuild 录入一次构建:先写 build_run(只 INSERT,INV-5)拿到 runID;
// 构建失败再接着录 failure_event(source=build, build_run_id=runID)并累加指纹索引。
// 成功构建只落 build_run,不录失败事件(PRD:只在失败的时候收)。
//
// 返回 build_run id、事件 id(仅失败录入时非零)与精确指纹(仅失败录入时非空,
// 供调用方按开关入队诊断,T11e/D-e3)。两步各自独立事务:
// build_run 写成而 failure_event 写败时,错误里带上 runID 便于排查。
func IngestBuild(ctx context.Context, s *store.Store, in BuildInput) (buildRunID, eventID int64, fp core.Fingerprint, err error) {
	if strings.TrimSpace(in.ProjectID) == "" {
		return 0, 0, "", errors.New("录入构建:缺少 project_id")
	}
	if strings.TrimSpace(in.CommitSha) == "" {
		return 0, 0, "", errors.New("录入构建:缺少 commit_sha(必填)")
	}
	if in.Status != StatusPassed && in.Status != StatusFailed {
		return 0, 0, "", fmt.Errorf("录入构建:status 应是 passed/failed,得到 %q", in.Status)
	}
	if in.StartedAt.IsZero() {
		return 0, 0, "", errors.New("录入构建:缺少 started_at")
	}
	combo := in.Combo
	if combo == nil {
		combo = core.VersionCombo{}
	}
	comboJSON, err := json.Marshal(combo)
	if err != nil {
		return 0, 0, "", fmt.Errorf("序列化版本组合: %w", err)
	}

	runID, err := s.Queries().InsertBuildRun(ctx, store.InsertBuildRunParams{
		ProjectID:      in.ProjectID,
		CommitSha:      in.CommitSha,
		CiRunUrl:       pgtype.Text{String: in.CiRunURL, Valid: in.CiRunURL != ""},
		StartedAt:      pgtype.Timestamptz{Time: in.StartedAt, Valid: true},
		FinishedAt:     pgtype.Timestamptz{Time: in.FinishedAt, Valid: !in.FinishedAt.IsZero()},
		ExitCode:       pgtype.Int4{Int32: int32(in.ExitCode), Valid: true},
		Status:         in.Status,
		Toolchain:      pgtype.Text{String: in.Toolchain, Valid: in.Toolchain != ""},
		ArtifactDigest: pgtype.Text{String: in.ArtifactDigest, Valid: in.ArtifactDigest != ""},
		VersionCombo:   comboJSON,
	})
	if err != nil {
		return 0, 0, "", fmt.Errorf("写 build_run: %w", err)
	}

	if in.Status != StatusFailed {
		return runID, 0, "", nil // 成功构建:只落 build_run
	}

	in.Input.Source = "build"
	in.Input.BuildRunID = runID
	eventID, fp, err = RecordFailure(ctx, s, in.Input)
	if err != nil {
		return runID, 0, "", fmt.Errorf("build_run=%d 已写,但录入失败事件出错: %w", runID, err)
	}
	return runID, eventID, fp, nil
}
