package runstore

// P1-33 适配器测试。
//
// 两层,同一文件(Allowed paths 只给了 runstore_test.go 一个测试文件):
//   - 纯单元层:行 ↔ 领域值的编码/重建、身份对账、API 面(AC1),
//     不需要数据库,`go test ./internal/runstore/...` 无 DB 也全绿;
//   - 数据库层:完整生命周期 round trip + 触发器拒绝路径(AC1/AC2/AC3 的
//     数据库侧防线)。每个测试独立 schema、自跑 goose 迁移、cleanup DROP
//     (与 internal/store 的 TestDB 同一隔离策略);数据库不可达时 SKIP,
//     不产出假失败。
//
// 领域校验用 errors.Is 判因,不做字符串匹配。

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"tally/internal/changeset"
	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/measurement"
	"tally/internal/store"
	"tally/internal/verification"
	"tally/migrations"
)

const runstoreTestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

// ---- fixtures ----

var (
	fixtureAt   = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	fixtureAt2  = time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC)
	fixtureAt3  = time.Date(2026, 1, 15, 11, 30, 0, 0, time.UTC)
	fixtureHash = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func testRecorder() evidence.Recorder {
	return evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "visible-exec-1"}
}

func testChangeSetFiles(tag string) []changeset.ChangedFile {
	return []changeset.ChangedFile{
		{Path: "internal/foo/" + tag + ".go", Content: "package foo // " + tag, Hunks: []changeset.Hunk{{OldStart: 1, OldLines: 2, NewStart: 1, NewLines: 3}}},
		{Path: "cmd/tool/" + tag + ".sh", Content: "#!/bin/sh\n", Executable: true},
	}
}

func mustChangeSet(t *testing.T, project core.ProjectID, base, candidate, tag string, files []changeset.ChangedFile, entities []string, producerRun int64) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           project,
		BaseRevision:      base,
		CandidateRevision: candidate,
		Files:             files,
		Entities:          entities,
		ProducerRun:       strconv.FormatInt(producerRun, 10),
		CreatedAt:         fixtureAt,
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	return cs
}

func int64Ptr(v int64) *int64 { return &v }
func int32Ptr(v int32) *int32 { return &v }

// ---- 测试库:每测试独立 schema,自跑迁移,cleanup DROP;不可达即 SKIP ----

func testStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = runstoreTestDSN
	}
	if !dbReachable(base) {
		t.Skipf("runstore 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_rs_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	setSearchPath := func(ctx context.Context, c *pgconn.PgConn) error {
		_, err := c.Exec(ctx, "SET search_path = "+pgx.Identifier{schema}.Sanitize()).ReadAll()
		return err
	}

	boot, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL: %v", err)
	}
	boot.MaxConns = 1
	boot.ConnConfig.Config.AfterConnect = setSearchPath
	bootDB := stdlib.OpenDB(*boot.ConnConfig)
	if _, err := bootDB.ExecContext(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		bootDB.Close()
		t.Fatalf("建测试 schema %s: %v", schema, err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, bootDB, migrations.FS)
	if err != nil {
		bootDB.Close()
		t.Fatalf("创建 goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		bootDB.Close()
		t.Fatalf("在测试 schema %s 跑迁移: %v", schema, err)
	}
	bootDB.Close()

	cfg, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL(连接池): %v", err)
	}
	cfg.ConnConfig.Config.AfterConnect = setSearchPath
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("建立测试连接池: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping 测试库: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		dropSchema(base, schema)
	})
	return New(store.New(pool)), pool
}

func dbReachable(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func dropSchema(base, schema string) {
	u, err := url.Parse(base)
	if err != nil {
		return
	}
	q := u.Query()
	q.Del("search_path")
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
}

// startTestRun 开一个测试 CodegenRun(每次调用都是新 Run:producer_run_id UNIQUE)。
func startTestRun(t *testing.T, ctx context.Context, st *Store, project core.ProjectID, task string) int64 {
	t.Helper()
	id, err := st.StartCodegenRun(ctx, StartRunSpec{
		Project:                   project,
		Task:                      task,
		Role:                      "codegen",
		Runtime:                   "fake",
		RuntimeVersion:            "fake-v1",
		Model:                     "test-model",
		BehaviorRevision:          "behavior-1",
		CostPlane:                 measurement.CostPlaneExecution,
		BaseCommit:                "base123",
		TaskRevision:              "sha256:task",
		VerificationRevision:      "sha256:verif",
		ContextPackHash:           "sha256:pack",
		WorkspaceID:               "wt-" + task + "-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		RuntimeCapabilitySnapshot: json.RawMessage(`{"session_isolation":true,"conformance_class":"verification-isolation-v1"}`),
		StartedAt:                 fixtureAt,
	})
	if err != nil {
		t.Fatalf("StartCodegenRun: %v", err)
	}
	return id
}

// recordTestVerificationRun 为 (run, changeset) 记一次通过执行的 verification_run。
func recordTestVerificationRun(t *testing.T, ctx context.Context, st *Store, project core.ProjectID, task string, csID changeset.ChangeSetID) int64 {
	t.Helper()
	id, err := st.RecordVerificationRun(ctx, VerificationRunSpec{
		Project:           project,
		Task:              task,
		ContractRevision:  2,
		ContractHash:      fixtureHash,
		ChangeSetID:       csID,
		EnvName:           "ci",
		EnvDefinitionHash: "sha256:envdef",
		VersionCombo:      core.VersionCombo{"go": "1.27", "os": "linux"},
		Runtime:           "fake",
		StartedAt:         fixtureAt,
		FinishedAt:        fixtureAt2,
		Status:            VerificationPass,
	})
	if err != nil {
		t.Fatalf("RecordVerificationRun: %v", err)
	}
	return id
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// mustModifyRejected 断言一条改写/删除语句被 append-only 触发器拒绝(AC1/AC3 的
// 数据库侧防线;适配器无 API 只是第一道防线)。
func mustModifyRejected(t *testing.T, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	_, err := pool.Exec(context.Background(), query, args...)
	if err == nil {
		t.Fatalf("append-only 触发器未拦截:%s", query)
	}
}

// ---- 纯单元层(无 DB) ----

func TestChangedFilesJSONRoundTrip(t *testing.T) {
	files := testChangeSetFiles("a")
	data, err := encodeChangedFiles(files)
	if err != nil {
		t.Fatalf("encodeChangedFiles: %v", err)
	}
	// 形状合同:JSON 数组;空集合编码为 []。
	if !strings.HasPrefix(string(data), "[") {
		t.Fatalf("changed_files 必须 JSON 数组,得到 %s", data)
	}
	if empty, err := encodeChangedFiles(nil); err != nil || string(empty) != "[]" {
		t.Fatalf("空 changed_files 应编码为 [],得到 %s(%v)", empty, err)
	}
	got, err := decodeChangedFiles(data)
	if err != nil {
		t.Fatalf("decodeChangedFiles: %v", err)
	}
	if len(got) != len(files) {
		t.Fatalf("文件数不一致: %d != %d", len(got), len(files))
	}
	for i, f := range files {
		if got[i].Path != f.Path || got[i].Content != f.Content || got[i].Executable != f.Executable ||
			len(got[i].Hunks) != len(f.Hunks) {
			t.Fatalf("文件 %d 往返不一致: %+v != %+v", i, got[i], f)
		}
		for j, h := range f.Hunks {
			if got[i].Hunks[j] != h {
				t.Fatalf("文件 %d hunk %d 往返不一致: %+v != %+v", i, j, got[i].Hunks[j], h)
			}
		}
	}
	// JSON null(存量哨兵)按空处理。
	if files, err := decodeChangedFiles([]byte("null")); err != nil || files != nil {
		t.Fatalf("null 应解为 nil,得到 %v(%v)", files, err)
	}
}

func TestChangeSetRowReconstruction(t *testing.T) {
	cs := mustChangeSet(t, "proj", "base1", "", "a", testChangeSetFiles("a"), []string{"svc/a"}, 7)
	files, err := encodeChangedFiles(cs.Files())
	if err != nil {
		t.Fatalf("encode files: %v", err)
	}
	entities, err := encodeStrings(cs.Entities())
	if err != nil {
		t.Fatalf("encode entities: %v", err)
	}
	row := store.Changeset{
		ID:              string(cs.ID()),
		ProjectID:       string(cs.Project()),
		BaseCommit:      cs.BaseRevision(),
		CandidateCommit: textPtr(""),
		PatchHash:       textPtr(cs.PatchIdentity()),
		ChangedFiles:    files,
		Entities:        entities,
		ProducerRunID:   7,
		CreatedAt:       tsPtr(fixtureAt),
	}

	got, err := changeSetFromRow(row)
	if err != nil {
		t.Fatalf("changeSetFromRow: %v", err)
	}
	if got.ID() != cs.ID() || got.Project() != cs.Project() || got.BaseRevision() != cs.BaseRevision() ||
		got.CandidateRevision() != cs.CandidateRevision() || got.PatchIdentity() != cs.PatchIdentity() ||
		got.ProducerRun() != cs.ProducerRun() || !got.CreatedAt().Equal(cs.CreatedAt()) {
		t.Fatalf("重建值不一致: got id=%s base=%s run=%s", got.ID(), got.BaseRevision(), got.ProducerRun())
	}
	gf, ge := got.Files(), got.Entities()
	// 文件按路径排序归一:cmd/tool/a.sh 在前。
	if len(gf) != 2 || len(ge) != 1 || ge[0] != "svc/a" || gf[0].Path != "cmd/tool/a.sh" || gf[0].Executable != true {
		t.Fatalf("重建 files/entities 不一致: %+v %+v", gf, ge)
	}

	// patch 身份被改:重建对账必须报错(AC3:不把矛盾值当真)。
	tampered := row
	tampered.PatchHash = textPtr("sha256:deadbeef")
	if _, err := changeSetFromRow(tampered); err == nil {
		t.Fatal("patch_hash 与内容不符应报错")
	}
	// 快照 id 被改:同一 ID 重推不一致,必须报错。
	tampered = row
	tampered.ID = "sha256:not-the-same"
	if _, err := changeSetFromRow(tampered); err == nil {
		t.Fatal("快照 id 与内容不符应报错")
	}
	// 改动文件内容被改:patch 身份对账同样必须失败。
	tampered = row
	tampered.ChangedFiles = []byte(strings.Replace(string(files), "package foo", "package evil", 1))
	if _, err := changeSetFromRow(tampered); err == nil {
		t.Fatal("changed_files 与 patch 身份不符应报错")
	}
}

func TestVerificationRunRowReconstruction(t *testing.T) {
	row := store.VerificationRun{
		ID: 3, ProjectID: "proj", TaskID: "T-1",
		ContractRevision: 2, ContractHash: fixtureHash, ChangesetID: "sha256:cs",
		EnvName: "ci", EnvDefinitionHash: "sha256:envdef",
		VersionCombo: []byte(`{"go":"1.27"}`), Runtime: "fake",
		StartedAt: tsPtr(fixtureAt), FinishedAt: pgtype.Timestamptz{}, Status: "pass",
	}
	got, err := verificationRunFromRow(row)
	if err != nil {
		t.Fatalf("verificationRunFromRow: %v", err)
	}
	if got.VersionCombo["go"] != "1.27" || got.Status != VerificationPass || got.ChangeSetID != "sha256:cs" || !got.FinishedAt.IsZero() {
		t.Fatalf("重建不一致: %+v", got)
	}
	bad := row
	bad.VersionCombo = []byte(`not-json`)
	if _, err := verificationRunFromRow(bad); err == nil {
		t.Fatal("version_combo 非法 JSON 应报错")
	}
	bad = row
	bad.Status = "green"
	if _, err := verificationRunFromRow(bad); err == nil {
		t.Fatal("status 越出 CHECK 取值应报错")
	}
}

func TestEvidenceRowReconstruction(t *testing.T) {
	run := &VerificationRun{ContractHash: fixtureHash, ChangeSetID: changeset.ChangeSetID("sha256:cs")}
	wantScope := evidence.Scope{
		Project:      "proj",
		Task:         "T-1",
		Acceptance:   "AC1",
		Verification: evidence.VerificationRevision(fixtureHash),
		ChangeSet:    evidence.ChangeSetID("sha256:cs"),
	}
	cmd := &command.CommandSpec{Argv: []string{"go", "test", "./..."}, Cwd: ".", Timeout: time.Minute}
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal command: %v", err)
	}
	row := store.VerificationEvidence{
		ID: 9, FactID: "f-1", RunID: 3, ProjectID: "proj", TaskID: "T-1",
		AcceptanceID: "AC1", BindingID: "b-1", VisibilityPolicy: "visible",
		Verdict: "pass", RecorderKind: "verification_run", RecorderActor: "visible-exec-1",
		Command: cmdJSON, ExitCode: pgtype.Int4{Int32: 0, Valid: true}, StdoutDigest: textPtr("sha256:out"),
		RecordedAt: tsPtr(fixtureAt),
	}

	got, err := evidenceFromRow(row, run)
	if err != nil {
		t.Fatalf("evidenceFromRow: %v", err)
	}
	if got.Scope != wantScope || got.Verdict != evidence.VerdictPass || got.BindingID != "b-1" ||
		got.VisibilityPolicy != verification.VisibilityVisible || got.ExitCode == nil || *got.ExitCode != 0 ||
		got.Command == nil {
		t.Fatalf("重建不一致: %+v", got)
	}
	if got.Command.Argv[1] != "test" || got.Command.Timeout != time.Minute {
		t.Fatalf("command 重建不一致: %+v", got.Command)
	}
	fact, err := got.Fact()
	if err != nil {
		t.Fatalf("Fact(): %v", err)
	}
	if fact.ID() != "f-1" || fact.Scope() != wantScope || !fact.IsValid() {
		t.Fatalf("领域 fact 重建不一致: %+v", fact)
	}

	// visibility_policy 越界。
	bad := row
	bad.VisibilityPolicy = "public"
	if _, err := evidenceFromRow(bad, run); err == nil {
		t.Fatal("visibility_policy 非法应报错")
	}
	// verdict 越界(重建即验证,不把矛盾值当真)。
	bad = row
	bad.Verdict = "green"
	if _, err := evidenceFromRow(bad, run); err == nil {
		t.Fatal("verdict 非法应报错")
	}
	// recorder 缺 actor。
	bad = row
	bad.RecorderActor = ""
	if _, err := evidenceFromRow(bad, run); err == nil {
		t.Fatal("recorder actor 为空应报错")
	}
	// command JSON 破坏。
	bad = row
	bad.Command = []byte(`{"argv":"not-a-list"}`)
	if _, err := evidenceFromRow(bad, run); err == nil {
		t.Fatal("command JSON 形状破坏应报错")
	}
	// JSON null command = 未记录,合法。
	bad = row
	bad.Command = []byte("null")
	if nulled, err := evidenceFromRow(bad, run); err != nil || nulled.Command != nil {
		t.Fatalf("null command 应重建为 nil,得到 %+v(%v)", nulled.Command, err)
	}
}

func TestExposureRowReconstruction(t *testing.T) {
	detail, err := encodeExposureDetail("canary-hit")
	if err != nil {
		t.Fatalf("encodeExposureDetail: %v", err)
	}
	row := store.EvidenceExposure{
		ID: 12, ProjectID: "proj", TaskID: "T-1", ContractRevision: 2,
		ContractHash: fixtureHash, BindingID: "b-9", AcceptanceID: textPtr("AC1"),
		ChangesetID: textPtr("sha256:cs"), Channel: "knowledge_index",
		Detail: detail, ExposedAt: tsPtr(fixtureAt),
	}
	got, err := exposureFromRow(row)
	if err != nil {
		t.Fatalf("exposureFromRow: %v", err)
	}
	// 领域身份 = serial 十进制串。
	if got.ID() != "12" {
		t.Fatalf("exposure id 应为 serial 串 12,得到 %s", got.ID())
	}
	want := evidence.BindingRef{Project: "proj", Verification: evidence.VerificationRevision(fixtureHash), BindingID: "b-9"}
	if got.Binding() != want || got.Channel() != evidence.ChannelKnowledgeIndex || got.Reason() != "canary-hit" {
		t.Fatalf("重建不一致: binding=%+v channel=%s reason=%s", got.Binding(), got.Channel(), got.Reason())
	}
	if acc, ok := got.Acceptance(); !ok || acc != "AC1" {
		t.Fatalf("acceptance 重建不一致: %s %v", acc, ok)
	}
	if cs, ok := got.ChangeSet(); !ok || cs != "sha256:cs" {
		t.Fatalf("changeset 锚重建不一致: %s %v", cs, ok)
	}
	if !got.BindingWide() {
		t.Fatal("knowledge_index 是 persistent 通道,BindingWide 应为 true")
	}
	// detail NULL = 无 reason,合法。
	row.Detail = nil
	if got, err := exposureFromRow(row); err != nil || got.Reason() != "" {
		t.Fatalf("null detail 应重建空 reason,得到 %q(%v)", got.Reason(), err)
	}
}

// TestStoreAPIHasNoFactMutators 机械固化 AC1/AC2 的 API 面:事实面不存在任何
// update/delete API;codegen_run 的两条 run-control 迁移是仅有的变更入口。
func TestStoreAPIHasNoFactMutators(t *testing.T) {
	allowedMutations := map[string]bool{
		"StartCodegenRun":       true, // 过程记录开跑(insert)
		"UpdateRunProgress":     true, // AC2:run-control,谓词限定 running
		"FinishCodegenRun":      true, // AC2:终态一次性迁移
		"SaveChangeSet":         true, // insert-once,幂等
		"AppendSupersededFact":  true, // append-only
		"RecordVerificationRun": true, // append-only
		"RecordEvidence":        true, // append-only
		"AppendEvidenceRuling":  true, // append-only
		"RecordExposure":        true, // append-only
	}
	typ := reflect.TypeOf(&Store{})
	seen := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		seen[name] = true
		for _, banned := range []string{"Delete", "Rewrite", "Truncate", "Drop"} {
			if strings.HasPrefix(name, banned) {
				t.Errorf("事实面出现删除/改写 API:%s(AC1)", name)
			}
		}
		if strings.HasPrefix(name, "Update") && name != "UpdateRunProgress" {
			t.Errorf("codegen_run 进度之外出现更新 API:%s(AC1/AC2)", name)
		}
	}
	for name := range allowedMutations {
		if !seen[name] {
			t.Errorf("缺少 append-only/run-control 写入口:%s", name)
		}
	}
}

// ---- 数据库层(不可达即 SKIP) ----

func TestCodegenRunLifecycleDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()

	id, err := st.StartCodegenRun(ctx, StartRunSpec{
		Project: "proj", Task: "T-life", Role: "codegen", Runtime: "fake",
		RuntimeVersion: "fake-v1", Model: "m", BehaviorRevision: "b1",
		CostPlane: measurement.CostPlaneControl, BaseCommit: "c0",
		TaskRevision: "sha256:t", VerificationRevision: "sha256:v", ContextPackHash: "sha256:cp",
		WorkspaceID: "wt-1", AgentSessionID: "sess-1",
		RuntimeCapabilitySnapshot: json.RawMessage(`{"session_isolation":true}`),
		EscalatedFrom:             0, StartedAt: fixtureAt,
	})
	if err != nil {
		t.Fatalf("StartCodegenRun: %v", err)
	}
	if id <= 0 {
		t.Fatalf("Run id 应为正,得到 %d", id)
	}

	run, err := st.CodegenRun(ctx, id)
	if err != nil {
		t.Fatalf("CodegenRun: %v", err)
	}
	// jsonb 会规范化空白,按 JSON 值比较快照而不是按字节。
	var gotSnap, wantSnap any
	if err := json.Unmarshal(run.RuntimeCapabilitySnapshot, &gotSnap); err != nil {
		t.Fatalf("snapshot 非法 JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(`{"session_isolation":true}`), &wantSnap); err != nil {
		t.Fatalf("want snapshot 非法 JSON: %v", err)
	}
	if run.Status != RunRunning || !run.StartedAt.Equal(fixtureAt) || !run.FinishedAt.IsZero() ||
		run.AgentSessionID != "sess-1" || run.EscalatedFrom != 0 || run.InputTokens != nil ||
		!reflect.DeepEqual(gotSnap, wantSnap) {
		t.Fatalf("开跑读取不一致: %+v", run)
	}

	ok, err := st.UpdateRunProgress(ctx, RunProgress{ID: id, Iterations: 2, InputTokens: int64Ptr(120), OutputTokens: int64Ptr(45), CostMicros: int64Ptr(7)})
	if err != nil || !ok {
		t.Fatalf("UpdateRunProgress: ok=%v err=%v", ok, err)
	}
	if run, err = st.CodegenRun(ctx, id); err != nil {
		t.Fatalf("CodegenRun: %v", err)
	}
	if run.Iterations != 2 || run.InputTokens == nil || *run.InputTokens != 120 {
		t.Fatalf("进度未落库: %+v", run)
	}

	ok, err = st.FinishCodegenRun(ctx, FinishRunSpec{
		ID: id, Status: RunCompleted, FinishedAt: fixtureAt2, Iterations: 3,
		InputTokens: int64Ptr(200), OutputTokens: int64Ptr(90), CostMicros: int64Ptr(15),
		ResultPatchHash: "sha256:patch", AgentClaim: "claim", ValidationResult: "ok",
	})
	if err != nil || !ok {
		t.Fatalf("FinishCodegenRun: ok=%v err=%v", ok, err)
	}
	if run, err = st.CodegenRun(ctx, id); err != nil {
		t.Fatalf("CodegenRun: %v", err)
	}
	if run.Status != RunCompleted || !run.FinishedAt.Equal(fixtureAt2) || run.Iterations != 3 ||
		run.ResultPatchHash != "sha256:patch" || run.AgentClaim != "claim" || run.FailureReason != "" {
		t.Fatalf("终态读取不一致: %+v", run)
	}

	// 终态一次性:二次迁移命中 0 行;running 之后的进度上报同样必须放弃(AC2)。
	if ok, err = st.FinishCodegenRun(ctx, FinishRunSpec{ID: id, Status: RunFailed, FailureReason: "rewrite"}); err != nil || ok {
		t.Fatalf("二次终态迁移应返回 false,得到 ok=%v err=%v", ok, err)
	}
	if ok, err = st.UpdateRunProgress(ctx, RunProgress{ID: id, Iterations: 9}); err != nil || ok {
		t.Fatalf("终态后的进度上报应返回 false,得到 ok=%v err=%v", ok, err)
	}
	if _, err := st.FinishCodegenRun(ctx, FinishRunSpec{ID: id, Status: RunRunning}); err == nil {
		t.Fatal("终态迁移目标为 running 应报错")
	}

	// 未知 id:命中 0 行,不是错误状态。
	if ok, err := st.FinishCodegenRun(ctx, FinishRunSpec{ID: 987654, Status: RunFailed, FinishedAt: fixtureAt2}); err != nil || ok {
		t.Fatalf("未知 id 终态迁移应 (false, nil),得到 ok=%v err=%v", ok, err)
	}
	if _, err := st.CodegenRun(ctx, 987654); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("未知 id 应包 pgx.ErrNoRows,得到 %v", err)
	}

	runs, err := st.CodegenRunsForTask(ctx, "proj", "T-life")
	if err != nil || len(runs) != 1 || runs[0].ID != id {
		t.Fatalf("CodegenRunsForTask 不一致: %v %+v", err, runs)
	}

	// 非法开跑输入在适配器即拒绝(不碰库)。
	validStart := StartRunSpec{
		Project: "proj", Task: "T-life", Role: "codegen", Runtime: "fake",
		RuntimeVersion: "fake-v1", Model: "m", BehaviorRevision: "b1",
		CostPlane: measurement.CostPlaneControl, BaseCommit: "c0",
		TaskRevision: "sha256:t", VerificationRevision: "sha256:v", ContextPackHash: "sha256:cp",
		WorkspaceID: "wt-1", RuntimeCapabilitySnapshot: json.RawMessage(`{"session_isolation":true}`),
	}
	bad := []StartRunSpec{validStart, validStart, validStart, validStart}
	bad[0].CostPlane = measurement.CostPlane("galactic")
	bad[1].RuntimeCapabilitySnapshot = json.RawMessage(`not-json`)
	bad[2].RuntimeCapabilitySnapshot = json.RawMessage(`[1,2]`) // 必须是 JSON 对象
	bad[3].Model = ""
	for i, spec := range bad {
		if _, err := st.StartCodegenRun(ctx, spec); err == nil {
			t.Fatalf("非法开跑输入 %d 应报错", i)
		}
	}
	if n := countRows(t, pool, "SELECT count(*) FROM codegen_run WHERE project_id='proj' AND task_id='T-life'"); n != 1 {
		t.Fatalf("非法输入不应落行,实际 %d 行", n)
	}
}

// ---- P0-02:observed 身份事实的编码与落库 ---------------------------------------

// observedTestSpec 是 observed 身份测试的开跑输入基线(期望坐标齐全)。
func observedTestSpec() StartRunSpec {
	return StartRunSpec{
		Project: "proj", Task: "T-obs", Role: "codegen", Runtime: "codex",
		RuntimeVersion: "1", Model: "economical", BehaviorRevision: "b1",
		CostPlane: measurement.CostPlaneExecution, BaseCommit: "c0",
		TaskRevision: "sha256:t", VerificationRevision: "sha256:v", ContextPackHash: "sha256:cp",
		WorkspaceID:               "wt-obs",
		RuntimeCapabilitySnapshot: json.RawMessage(`{"name":"codex","version":"1","capabilities":["model.select"]}`),
		StartedAt:                 fixtureAt,
	}
}

func TestAttachObservedIdentityEncoding(t *testing.T) {
	base := json.RawMessage(`{"name":"codex","version":"1","capabilities":["model.select"]}`)

	// 全部未观察:原样返回(字节不动,既有调用方行为不变)。
	out, err := attachObservedIdentity(base, "", "", "")
	if err != nil {
		t.Fatalf("attachObservedIdentity(零值): %v", err)
	}
	if string(out) != string(base) {
		t.Fatalf("未观察时快照应原样返回: %s", out)
	}

	// 有观察事实:注入顶层 observed_identity,调用方其余键保留。
	out, err = attachObservedIdentity(base, "codex-cli", "0.153.0", "")
	if err != nil {
		t.Fatalf("attachObservedIdentity: %v", err)
	}
	var top map[string]any
	if err := json.Unmarshal(out, &top); err != nil {
		t.Fatalf("注入后不是 JSON 对象: %v", err)
	}
	if top["name"] != "codex" || top["version"] != "1" {
		t.Fatalf("调用方原有键被破坏: %s", out)
	}
	obs, ok := top["observed_identity"].(map[string]any)
	if !ok {
		t.Fatalf("缺少 observed_identity: %s", out)
	}
	// 模型未观察:omitempty,不落假空值(未观察 ≠ 空观察)。
	if obs["runtime_name"] != "codex-cli" || obs["runtime_version"] != "0.153.0" {
		t.Fatalf("observed_identity 编码不符: %s", out)
	}
	if _, has := obs["model"]; has {
		t.Fatalf("未观察的 model 不应出现: %s", out)
	}

	// 输出确定:同输入两次编码字节一致(map 键排序)。
	out2, err := attachObservedIdentity(base, "codex-cli", "0.153.0", "")
	if err != nil || string(out2) != string(out) {
		t.Fatalf("编码应确定: %s vs %s (%v)", out, out2, err)
	}
}

func TestStartRunSpecRejectsReservedObservedIdentityKey(t *testing.T) {
	// observed_identity 键由本包按 observed_* 输入注入;调用方预置即拒绝
	//(防两类身份事实来源混淆,Decision Lock 15)。
	spec := observedTestSpec()
	spec.RuntimeCapabilitySnapshot = json.RawMessage(`{"observed_identity":{"model":"x"}}`)
	if err := spec.validate(); err == nil {
		t.Fatal("预置 observed_identity 顶层键应被 validate 拒绝")
	}
	// 合法形状 + observed 输入仍通过(开跑校验不因观察输入收紧)。
	ok := observedTestSpec()
	ok.ObservedRuntimeName = "codex-cli"
	ok.ObservedRuntimeVersion = "0.153.0"
	if err := ok.validate(); err != nil {
		t.Fatalf("合法 observed 输入不应报错: %v", err)
	}
}

func TestObservedIdentityPersistedWithRunDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()

	// 带 observed 事实开跑:AgentSessionID 有独立列;version/model 无列,
	// 落在快照 JSONB 的 observed_identity 键里,与期望坐标同存可区分。
	spec := observedTestSpec()
	spec.WorkspaceID = "wt-obs-1"
	spec.AgentSessionID = "thread-obs-1"
	spec.ObservedRuntimeName = "codex-cli"
	spec.ObservedRuntimeVersion = "0.153.0"
	// ObservedModel 刻意留空:runtime 未自报模型 = 未观察,不落假数据。
	id, err := st.StartCodegenRun(ctx, spec)
	if err != nil {
		t.Fatalf("StartCodegenRun(observed): %v", err)
	}
	run, err := st.CodegenRun(ctx, id)
	if err != nil {
		t.Fatalf("CodegenRun: %v", err)
	}
	if run.AgentSessionID != "thread-obs-1" {
		t.Fatalf("agent_session_id = %q, want thread-obs-1", run.AgentSessionID)
	}
	// 期望坐标列不被观察值覆盖:两类事实同存、可区分(P0-02 AC1)。
	if run.RuntimeVersion != "1" || run.Model != "economical" || run.Runtime != "codex" {
		t.Fatalf("期望坐标被覆盖: runtime=%s runtime_version=%s model=%s", run.Runtime, run.RuntimeVersion, run.Model)
	}
	var top map[string]any
	if err := json.Unmarshal(run.RuntimeCapabilitySnapshot, &top); err != nil {
		t.Fatalf("落库快照不是 JSON 对象: %v", err)
	}
	obs, ok := top["observed_identity"].(map[string]any)
	if !ok {
		t.Fatalf("落库快照缺少 observed_identity: %s", run.RuntimeCapabilitySnapshot)
	}
	if obs["runtime_name"] != "codex-cli" || obs["runtime_version"] != "0.153.0" {
		t.Fatalf("落库 observed_identity 不符: %s", run.RuntimeCapabilitySnapshot)
	}
	if _, has := obs["model"]; has {
		t.Fatalf("未观察模型不得落库为观察事实: %s", run.RuntimeCapabilitySnapshot)
	}
	if top["name"] != "codex" || top["version"] != "1" {
		t.Fatalf("声明能力快照原有键被破坏: %s", run.RuntimeCapabilitySnapshot)
	}

	// 无 observed 输入的既有路径:快照原样落库,agent_session_id 为 NULL。
	legacy := observedTestSpec()
	legacy.Task = "T-obs-legacy"
	legacy.WorkspaceID = "wt-obs-2"
	id2, err := st.StartCodegenRun(ctx, legacy)
	if err != nil {
		t.Fatalf("StartCodegenRun(legacy): %v", err)
	}
	run2, err := st.CodegenRun(ctx, id2)
	if err != nil {
		t.Fatalf("CodegenRun(legacy): %v", err)
	}
	var gotLegacy, wantLegacy any
	if err := json.Unmarshal(run2.RuntimeCapabilitySnapshot, &gotLegacy); err != nil {
		t.Fatalf("legacy 快照非法 JSON: %v", err)
	}
	if err := json.Unmarshal(legacy.RuntimeCapabilitySnapshot, &wantLegacy); err != nil {
		t.Fatalf("want 快照非法 JSON: %v", err)
	}
	if !reflect.DeepEqual(gotLegacy, wantLegacy) {
		t.Fatalf("无观察输入时快照应原样落库: %s vs %s", run2.RuntimeCapabilitySnapshot, legacy.RuntimeCapabilitySnapshot)
	}
	if run2.AgentSessionID != "" {
		t.Fatalf("legacy 路径 agent_session_id 应为空: %q", run2.AgentSessionID)
	}

	// 身份坐标开跑即写定:进度/终态 run-control 迁移不触碰快照与身份列(AC4)。
	if ok, err := st.UpdateRunProgress(ctx, RunProgress{ID: id, Iterations: 1}); err != nil || !ok {
		t.Fatalf("UpdateRunProgress: ok=%v err=%v", ok, err)
	}
	if ok, err := st.FinishCodegenRun(ctx, FinishRunSpec{ID: id, Status: RunCompleted, FinishedAt: fixtureAt2, Iterations: 1}); err != nil || !ok {
		t.Fatalf("FinishCodegenRun: ok=%v err=%v", ok, err)
	}
	after, err := st.CodegenRun(ctx, id)
	if err != nil {
		t.Fatalf("CodegenRun(after): %v", err)
	}
	if after.Status != RunCompleted || after.AgentSessionID != "thread-obs-1" || after.Model != "economical" {
		t.Fatalf("run-control 迁移不应改写身份事实: %+v", after)
	}
	var afterTop map[string]any
	if err := json.Unmarshal(after.RuntimeCapabilitySnapshot, &afterTop); err != nil {
		t.Fatalf("迁移后快照非法 JSON: %v", err)
	}
	if _, ok := afterTop["observed_identity"]; !ok {
		t.Fatalf("迁移后 observed_identity 应仍在: %s", after.RuntimeCapabilitySnapshot)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM codegen_run WHERE project_id='proj' AND task_id='T-obs'"); n != 1 {
		t.Fatalf("T-obs 行数 = %d, want 1", n)
	}
}

func TestChangeSetSaveOnceAndReadBackDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-cs"

	runA := startTestRun(t, ctx, st, project, task)
	cs := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), []string{"svc/a"}, runA)
	if err := st.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("SaveChangeSet: %v", err)
	}

	got, err := st.ChangeSet(ctx, cs.ID())
	if err != nil {
		t.Fatalf("ChangeSet: %v", err)
	}
	if got.ID() != cs.ID() || got.Project() != cs.Project() || got.BaseRevision() != cs.BaseRevision() ||
		got.PatchIdentity() != cs.PatchIdentity() || got.ProducerRun() != cs.ProducerRun() ||
		!got.CreatedAt().Equal(cs.CreatedAt()) {
		t.Fatalf("读回不一致: id=%s patch=%s run=%s", got.ID(), got.PatchIdentity(), got.ProducerRun())
	}
	gf, ge := got.Files(), got.Entities()
	wantF := cs.Files()
	if len(gf) != len(wantF) || len(ge) != 1 || ge[0] != "svc/a" {
		t.Fatalf("files/entities 读回不一致: %+v %+v", gf, ge)
	}
	for i := range wantF {
		if gf[i].Path != wantF[i].Path || gf[i].Content != wantF[i].Content || gf[i].Executable != wantF[i].Executable {
			t.Fatalf("file %d 读回不一致", i)
		}
	}

	// 同一快照重复保存:幂等忽略,不报错也不产生第二行(AC3 insert-once)。
	if err := st.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("重复 SaveChangeSet 应幂等: %v", err)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset WHERE id=$1", string(cs.ID())); n != 1 {
		t.Fatalf("重复保存产生 %d 行,应为 1", n)
	}
	list, err := st.ChangeSetsForTask(ctx, project, task)
	if err != nil || len(list) != 1 || list[0].ID() != cs.ID() {
		t.Fatalf("ChangeSetsForTask 不一致: %v %d", err, len(list))
	}

	// 纯 commit 候选(无改动文件):patch 身份为空,身份仍可重建。
	runB := startTestRun(t, ctx, st, project, task)
	cs2 := mustChangeSet(t, project, "base1", "cand-abc", "b", nil, nil, runB)
	if err := st.SaveChangeSet(ctx, cs2); err != nil {
		t.Fatalf("SaveChangeSet(纯 commit): %v", err)
	}
	got2, err := st.ChangeSet(ctx, cs2.ID())
	if err != nil {
		t.Fatalf("ChangeSet(纯 commit): %v", err)
	}
	if got2.PatchIdentity() != "" || len(got2.Files()) != 0 || got2.CandidateRevision() != "cand-abc" {
		t.Fatalf("纯 commit 候选重建不一致: patch=%q files=%d", got2.PatchIdentity(), len(got2.Files()))
	}

	// producer run 引用不是数字 id:适配器拒绝。
	bad, err := changeset.New(changeset.Spec{Project: project, BaseRevision: "x", CandidateRevision: "y", ProducerRun: "not-a-number"})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	if err := st.SaveChangeSet(ctx, bad); err == nil {
		t.Fatal("非数字 producer run 应报错")
	}
}

func TestChangeSetImmutableAtDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()

	runID := startTestRun(t, ctx, st, "proj", "T-immutable")
	cs := mustChangeSet(t, "proj", "base1", "", "a", testChangeSetFiles("a"), nil, runID)
	if err := st.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("SaveChangeSet: %v", err)
	}

	mustModifyRejected(t, pool, "UPDATE changeset SET base_commit='evil' WHERE id=$1", string(cs.ID()))
	mustModifyRejected(t, pool, "DELETE FROM changeset WHERE id=$1", string(cs.ID()))
}

func TestSupersededFactAppendOnlyDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-supersede"

	runA := startTestRun(t, ctx, st, project, task)
	csA := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), nil, runA)
	runB := startTestRun(t, ctx, st, project, task)
	csB := mustChangeSet(t, project, "base1", "", "b", testChangeSetFiles("b"), nil, runB)
	if err := st.SaveChangeSet(ctx, csA); err != nil {
		t.Fatalf("SaveChangeSet a: %v", err)
	}
	if err := st.SaveChangeSet(ctx, csB); err != nil {
		t.Fatalf("SaveChangeSet b: %v", err)
	}

	fact, err := changeset.NewSupersededFact(csA.ID(), csB.ID(), changeset.SupersedeRework, fixtureAt)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := st.AppendSupersededFact(ctx, fact); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}

	got, err := st.ChangeSetSupersessions(ctx, csA.ID())
	if err != nil || len(got) != 1 {
		t.Fatalf("ChangeSetSupersessions: %v %d", err, len(got))
	}
	if got[0].Old != csA.ID() || got[0].New != csB.ID() || got[0].Reason != changeset.SupersedeRework || !got[0].At.Equal(fixtureAt) {
		t.Fatalf("取代事实重建不一致: %+v", got[0])
	}
	// new 侧同样可查。
	if gotB, err := st.ChangeSetSupersessions(ctx, csB.ID()); err != nil || len(gotB) != 1 {
		t.Fatalf("new 侧取代事实: %v %d", err, len(gotB))
	}
	if all, err := st.AllChangeSetSupersessions(ctx); err != nil || len(all) != 1 {
		t.Fatalf("AllChangeSetSupersessions: %v %d", err, len(all))
	}

	// 同一事实重复追加:幂等去重;不同事实仍可各自成行。
	if err := st.AppendSupersededFact(ctx, fact); err != nil {
		t.Fatalf("重复追加应幂等: %v", err)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset_superseded_fact"); n != 1 {
		t.Fatalf("重复追加产生 %d 行,应为 1", n)
	}
	later, err := changeset.NewSupersededFact(csA.ID(), csB.ID(), changeset.SupersedeRework, fixtureAt3)
	if err != nil {
		t.Fatalf("NewSupersededFact(later): %v", err)
	}
	if err := st.AppendSupersededFact(ctx, later); err != nil {
		t.Fatalf("不同时刻取代应各自成行: %v", err)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset_superseded_fact"); n != 2 {
		t.Fatalf("不同时刻取代应 2 行,实际 %d", n)
	}

	// 非法事实(同 old/new、未知 reason、零时刻)在适配器即拒绝。
	if err := st.AppendSupersededFact(ctx, changeset.SupersededFact{Old: csA.ID(), New: csA.ID(), Reason: changeset.SupersedeRework, At: fixtureAt}); err == nil {
		t.Fatal("old==new 应报错")
	}
	if err := st.AppendSupersededFact(ctx, changeset.SupersededFact{Old: csA.ID(), New: csB.ID(), Reason: "because", At: fixtureAt}); err == nil {
		t.Fatal("未知 reason 应报错")
	}
	if err := st.AppendSupersededFact(ctx, changeset.SupersededFact{Old: csA.ID(), New: csB.ID(), Reason: changeset.SupersedeRework}); err == nil {
		t.Fatal("零时刻应报错")
	}

	mustModifyRejected(t, pool, "UPDATE changeset_superseded_fact SET reason='hax'")
	mustModifyRejected(t, pool, "DELETE FROM changeset_superseded_fact")
}

// P2-05(数据库层):闭环 LineageSink → runstore 的 rework 取代事实
// append-only round trip。Engine 的时序纪律(取代事实在复验之前落库)由
// closedloop 的测试守护;这里证明同一事实经持久化出口落库后可读回、
// 重复追加幂等、且数据库侧没有改写/删除路径。
func TestClosedLoopLineageSinkReworkSupersessionDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-closedloop-lineage"

	runA := startTestRun(t, ctx, st, project, task)
	csA := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), nil, runA)
	runB := startTestRun(t, ctx, st, project, task)
	csB := mustChangeSet(t, project, "base1", "", "b", testChangeSetFiles("b"), nil, runB)
	if err := st.SaveChangeSet(ctx, csA); err != nil {
		t.Fatalf("SaveChangeSet a: %v", err)
	}
	if err := st.SaveChangeSet(ctx, csB); err != nil {
		t.Fatalf("SaveChangeSet b: %v", err)
	}

	// 经 Store 原生 append-only API 直接落事实(closedloop.NewStoreLineageSink
	// 的适配行为由 closedloop 包自身的测试覆盖;这里测持久化事实面本身)。
	fact, err := changeset.NewSupersededFact(csA.ID(), csB.ID(), changeset.SupersedeRework, fixtureAt)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := st.AppendSupersededFact(ctx, fact); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}
	got, err := st.ChangeSetSupersessions(ctx, csA.ID())
	if err != nil || len(got) != 1 {
		t.Fatalf("ChangeSetSupersessions: %v %d", err, len(got))
	}
	if got[0].Old != csA.ID() || got[0].New != csB.ID() || got[0].Reason != changeset.SupersedeRework || !got[0].At.Equal(fixtureAt) {
		t.Fatalf("经闭环出口落库的取代事实不一致: %+v", got[0])
	}

	// 同一事实重复追加:幂等;事实面仍无改写/删除路径(触发器拒绝)。
	if err := st.AppendSupersededFact(ctx, fact); err != nil {
		t.Fatalf("重复追加应幂等: %v", err)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM changeset_superseded_fact"); n != 1 {
		t.Fatalf("重复追加产生 %d 行,应为 1", n)
	}
	mustModifyRejected(t, pool, "UPDATE changeset_superseded_fact SET reason='hax'")
	mustModifyRejected(t, pool, "DELETE FROM changeset_superseded_fact")
}

func TestVerificationRunRecordReadDB(t *testing.T) {
	st, _ := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-vr"

	runID := startTestRun(t, ctx, st, project, task)
	cs := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), nil, runID)
	if err := st.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("SaveChangeSet: %v", err)
	}

	spec := VerificationRunSpec{
		Project: project, Task: task, ContractRevision: 2, ContractHash: fixtureHash,
		ChangeSetID: cs.ID(), EnvName: "ci", EnvDefinitionHash: "sha256:envdef",
		VersionCombo: core.VersionCombo{"go": "1.27"}, Runtime: "fake",
		StartedAt: fixtureAt, FinishedAt: fixtureAt2, Status: VerificationPass,
	}
	vrID, err := st.RecordVerificationRun(ctx, spec)
	if err != nil {
		t.Fatalf("RecordVerificationRun: %v", err)
	}
	got, err := st.VerificationRun(ctx, project, vrID)
	if err != nil {
		t.Fatalf("VerificationRun: %v", err)
	}
	if got.ID != vrID || got.ContractRevision != 2 || got.ContractHash != fixtureHash ||
		got.ChangeSetID != cs.ID() || got.EnvName != "ci" || got.VersionCombo["go"] != "1.27" ||
		got.Status != VerificationPass || !got.StartedAt.Equal(fixtureAt) || !got.FinishedAt.Equal(fixtureAt2) {
		t.Fatalf("verification_run 读回不一致: %+v", got)
	}
	forCS, err := st.VerificationRunsForChangeSet(ctx, project, cs.ID())
	if err != nil || len(forCS) != 1 || forCS[0].ID != vrID {
		t.Fatalf("VerificationRunsForChangeSet: %v %d", err, len(forCS))
	}
	forTask, err := st.VerificationRunsForTask(ctx, project, task)
	if err != nil || len(forTask) != 1 {
		t.Fatalf("VerificationRunsForTask: %v %d", err, len(forTask))
	}

	// 同一 (contract revision, ChangeSet) 的再次执行各自成行(§8.7.2 全量留痕)。
	spec.Status = VerificationFail
	spec.FinishedAt = fixtureAt3
	second, err := st.RecordVerificationRun(ctx, spec)
	if err != nil {
		t.Fatalf("第二次执行应各自成行: %v", err)
	}
	if second == vrID {
		t.Fatal("两次执行不应共享行 id")
	}

	// 非法输入在适配器即拒绝。
	validSpec := VerificationRunSpec{
		Project: project, Task: task, ContractRevision: 2, ContractHash: fixtureHash,
		ChangeSetID: cs.ID(), EnvName: "ci", EnvDefinitionHash: "sha256:envdef",
		VersionCombo: core.VersionCombo{"go": "1"}, Runtime: "fake",
		StartedAt: fixtureAt, Status: VerificationPass,
	}
	bad := []VerificationRunSpec{validSpec, validSpec, validSpec, validSpec}
	bad[0].ContractRevision = 0
	bad[1].VersionCombo = nil
	bad[2].Status = VerificationStatus("green")
	bad[3].ChangeSetID = ""
	for i, spec := range bad {
		if _, err := st.RecordVerificationRun(ctx, spec); err == nil {
			t.Fatalf("非法 verification_run 输入 %d 应报错", i)
		}
	}
	if _, err := st.VerificationRun(ctx, project, 424242); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("未知 verification_run 应包 pgx.ErrNoRows,得到 %v", err)
	}
}

func TestEvidenceRecordRulingsAndApplicabilityDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-ev"

	runID := startTestRun(t, ctx, st, project, task)
	cs := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), nil, runID)
	if err := st.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("SaveChangeSet: %v", err)
	}
	vrID := recordTestVerificationRun(t, ctx, st, project, task, cs.ID())

	rec := EvidenceRecord{
		FactID:           "fact-pass-1",
		RunID:            vrID,
		Task:             task,
		Acceptance:       "AC1",
		BindingID:        "b-visible",
		VisibilityPolicy: verification.VisibilityVisible,
		Verdict:          evidence.VerdictPass,
		Recorder:         testRecorder(),
		Command:          &command.CommandSpec{Argv: []string{"go", "test", "./internal/runstore/..."}, Cwd: "."},
		ExitCode:         int32Ptr(0),
		StdoutDigest:     "sha256:out",
		RecordedAt:       fixtureAt,
	}
	evID, err := st.RecordEvidence(ctx, project, rec)
	if err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}

	got, err := st.Evidence(ctx, project, rec.FactID)
	if err != nil {
		t.Fatalf("Evidence: %v", err)
	}
	wantScope := evidence.Scope{
		Project:      project,
		Task:         evidence.TaskID(task),
		Acceptance:   "AC1",
		Verification: evidence.VerificationRevision(fixtureHash),
		ChangeSet:    evidence.ChangeSetID(cs.ID()),
	}
	if got.ID != evID || got.Scope != wantScope || got.Verdict != evidence.VerdictPass ||
		got.VisibilityPolicy != verification.VisibilityVisible || got.Recorder != testRecorder() ||
		got.ExitCode == nil || *got.ExitCode != 0 || got.Command == nil || len(got.Command.Argv) != 3 {
		t.Fatalf("evidence 读回不一致: %+v", got)
	}
	fact, err := got.Fact()
	if err != nil || fact.ID() != rec.FactID || fact.Scope() != wantScope || !fact.IsValid() {
		t.Fatalf("领域 fact 重建: %v %+v", err, fact)
	}

	// 同一 fact_id 二次记录 = 错误,不是更新(INV-FACT-01)。
	if _, err := st.RecordEvidence(ctx, project, rec); err == nil {
		t.Fatal("fact_id 重复应报错")
	}
	// agent 提案通道不能确认机器事实(INV-AUTH-01/02),适配器先于 DB CHECK 拒绝。
	agentRec := rec
	agentRec.FactID = "fact-agent"
	agentRec.Recorder = evidence.Recorder{Kind: evidence.RecorderAgent, Actor: "claude"}
	if _, err := st.RecordEvidence(ctx, project, agentRec); !errors.Is(err, evidence.ErrRecorderCannotRecordFact) {
		t.Fatalf("agent recorder 应被拒,得到 %v", err)
	}

	// 第二条:合法 FAIL,随后被裁定失效(合法 FAIL 不被后来的 PASS 抹掉,§8.8.1)。
	failRec := rec
	failRec.FactID = "fact-fail-1"
	failRec.Verdict = evidence.VerdictFail
	failRec.Command = nil
	failRec.ExitCode = int32Ptr(1)
	failRec.RecordedAt = fixtureAt2
	failID, err := st.RecordEvidence(ctx, project, failRec)
	if err != nil {
		t.Fatalf("RecordEvidence(fail): %v", err)
	}

	scope := wantScope
	ruling, err := evidence.NewRuling(evidence.RulingInvalidEvidence, "executor crashed mid-run", fixtureAt3, evidence.Recorder{Kind: evidence.RecorderHuman, Actor: "owner"})
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	if _, err := st.AppendEvidenceRuling(ctx, project, failID, ruling); err != nil {
		t.Fatalf("AppendEvidenceRuling: %v", err)
	}
	rulings, err := st.EvidenceRulings(ctx, project, failID)
	if err != nil || len(rulings) != 1 || rulings[0].Kind != evidence.RulingInvalidEvidence || rulings[0].Reason != "executor crashed mid-run" {
		t.Fatalf("EvidenceRulings: %v %+v", err, rulings)
	}

	// gate 输入:五维 scope 命中 f-pass-1,被裁定的 f-fail-1 排除(applicability×validity)。
	applicable, err := st.ApplicableValidEvidence(ctx, scope, 2)
	if err != nil {
		t.Fatalf("ApplicableValidEvidence: %v", err)
	}
	if len(applicable) != 1 || applicable[0].FactID != "fact-pass-1" {
		t.Fatalf("applicable valid evidence 应只剩 fact-pass-1,得到 %+v", applicable)
	}

	// 裁定通道不 fact-capable 不能使证据失效。
	agentRuling, err := evidence.NewRuling(evidence.RulingInvalidEvidence, "agent says so", fixtureAt3, evidence.Recorder{Kind: evidence.RecorderAgent, Actor: "a"})
	if err == nil {
		if _, err := st.AppendEvidenceRuling(ctx, project, failID, agentRuling); err == nil {
			t.Fatal("agent 裁定应被拒")
		}
	}
	// 空 reason 裁定被拒。
	empty, _ := evidence.NewRuling(evidence.RulingInvalidEvidence, "x", fixtureAt3, evidence.Recorder{Kind: evidence.RecorderHuman, Actor: "o"})
	empty.Reason = ""
	if _, err := st.AppendEvidenceRuling(ctx, project, failID, empty); err == nil {
		t.Fatal("空 reason 裁定应报错")
	}
	// 未知 evidence 的裁定被 FK 拒绝。
	if _, err := st.AppendEvidenceRuling(ctx, project, 987654, ruling); err == nil {
		t.Fatal("未知 evidence 的裁定应报错")
	}

	// held_out 可见性策略同样可落可读(执行时快照)。
	heldRec := rec
	heldRec.FactID = "fact-held-1"
	heldRec.BindingID = "b-held"
	heldRec.VisibilityPolicy = verification.VisibilityHeldOut
	heldRec.RawOutputRef = "holdout://run/3/out"
	if _, err := st.RecordEvidence(ctx, project, heldRec); err != nil {
		t.Fatalf("RecordEvidence(held_out): %v", err)
	}
	held, err := st.Evidence(ctx, project, "fact-held-1")
	if err != nil || held.VisibilityPolicy != verification.VisibilityHeldOut || held.RawOutputRef != "holdout://run/3/out" {
		t.Fatalf("held_out evidence 读回: %v %+v", err, held)
	}

	// 数据库侧 append-only 防线:适配器之外直接改写/删除必须被触发器拦截。
	mustModifyRejected(t, pool, "UPDATE verification_evidence SET verdict='pass' WHERE id=$1", failID)
	mustModifyRejected(t, pool, "DELETE FROM verification_evidence WHERE id=$1", failID)
	mustModifyRejected(t, pool, "UPDATE verification_run SET status='pass' WHERE id=$1", vrID)
	mustModifyRejected(t, pool, "DELETE FROM verification_evidence_ruling")
}

func TestExposureAppendOnlyDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-exp"

	runID := startTestRun(t, ctx, st, project, task)
	cs := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), nil, runID)
	if err := st.SaveChangeSet(ctx, cs); err != nil {
		t.Fatalf("SaveChangeSet: %v", err)
	}

	binding := evidence.BindingRef{Project: project, Verification: evidence.VerificationRevision(fixtureHash), BindingID: "b-held"}
	rec := ExposureRecord{
		Task:             task,
		ContractRevision: 2,
		Binding:          binding,
		Acceptance:       "AC1",
		Channel:          evidence.ChannelKnowledgeIndex,
		Reason:           "canary-hit",
		ExposedAt:        fixtureAt,
	}
	e, err := st.RecordExposure(ctx, rec)
	if err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}
	if e.ID() == "" {
		t.Fatal("落库后领域身份应由 DB serial 分配")
	}
	if e.Binding() != binding || e.Channel() != evidence.ChannelKnowledgeIndex || e.Reason() != "canary-hit" || !e.BindingWide() {
		t.Fatalf("暴露事实重建不一致: %+v", e)
	}
	if _, ok := e.ChangeSet(); ok {
		t.Fatal("未锚定暴露不应带 ChangeSet")
	}

	// 按 task / binding 两个投影读回,值与写入口返回一致。
	forTask, err := st.ExposuresForTask(ctx, project, task)
	if err != nil || len(forTask) != 1 || forTask[0] != e {
		t.Fatalf("ExposuresForTask: %v %d", err, len(forTask))
	}
	forBinding, err := st.ExposuresForBinding(ctx, project, fixtureHash, "b-held")
	if err != nil || len(forBinding) != 1 || forBinding[0] != e {
		t.Fatalf("ExposuresForBinding: %v %d", err, len(forBinding))
	}

	// 锚定 candidate 的 lineage 局部暴露。
	rec2 := rec
	rec2.Channel = evidence.ChannelDebugDisclosure
	rec2.Reason = ""
	rec2.ChangeSet = evidence.ChangeSetID(cs.ID())
	rec2.ExposedAt = fixtureAt2
	e2, err := st.RecordExposure(ctx, rec2)
	if err != nil {
		t.Fatalf("RecordExposure(anchored): %v", err)
	}
	if e2.BindingWide() {
		t.Fatal("debug_disclosure 是 lineage 通道,不应 BindingWide")
	}
	if anchored, ok := e2.ChangeSet(); !ok || anchored != evidence.ChangeSetID(cs.ID()) {
		t.Fatalf("锚定暴露应带 ChangeSet:%s %v", anchored, ok)
	}
	forCS, err := st.ExposuresForChangeSet(ctx, project, cs.ID())
	if err != nil {
		t.Fatalf("ExposuresForChangeSet: %v", err)
	}
	if len(forCS) != 1 || forCS[0].ID() != e2.ID() {
		t.Fatalf("anchored 投影应命中 1 条,得到 %d", len(forCS))
	}

	// 非法输入:binding 不完整 / channel 为空 / revision 非法。
	if _, err := st.RecordExposure(ctx, ExposureRecord{Task: task, ContractRevision: 2, Binding: evidence.BindingRef{Project: project}, Channel: evidence.ChannelManual, ExposedAt: fixtureAt}); err == nil {
		t.Fatal("binding 不完整应报错")
	}
	if _, err := st.RecordExposure(ctx, ExposureRecord{Task: task, ContractRevision: 2, Binding: binding, ExposedAt: fixtureAt}); err == nil {
		t.Fatal("channel 为空应报错")
	}
	if _, err := st.RecordExposure(ctx, ExposureRecord{Task: task, ContractRevision: 0, Binding: binding, Channel: evidence.ChannelManual, ExposedAt: fixtureAt}); err == nil {
		t.Fatal("revision 0 应报错")
	}

	mustModifyRejected(t, pool, "UPDATE evidence_exposure SET channel='hax'")
	mustModifyRejected(t, pool, "DELETE FROM evidence_exposure")
}

// ---- 取代 lineage 暴露投影(P0-07) ----

// lineageFactGraph 把取代事实按「参与方 → 事实」索引(与
// ChangeSetSupersessions 同合同:old 或 new 侧命中都返回),供纯遍历核测试。
func lineageFactGraph(t *testing.T, facts ...changeset.SupersededFact) func(changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
	t.Helper()
	byNode := map[changeset.ChangeSetID][]changeset.SupersededFact{}
	for _, f := range facts {
		byNode[f.Old] = append(byNode[f.Old], f)
		byNode[f.New] = append(byNode[f.New], f)
	}
	return func(id changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
		return byNode[id], nil
	}
}

// TestWalkSupersessionAncestorsDeterministic 固化 P0-07 AC1:祖先展开按层
// BFS、层内按身份排序、后代方向(old 侧)不参与、环与重复由 visited 截断,
// 输出只由事实集合决定,与事实返回顺序无关。
func TestWalkSupersessionAncestorsDeterministic(t *testing.T) {
	mk := func(old, new string, at time.Time) changeset.SupersededFact {
		t.Helper()
		f, err := changeset.NewSupersededFact(changeset.ChangeSetID(old), changeset.ChangeSetID(new), changeset.SupersedeRework, at)
		if err != nil {
			t.Fatalf("NewSupersededFact(%s→%s): %v", old, new, err)
		}
		return f
	}
	// 图:A→B1、A→B2(多父汇聚到 C);B1→C、B2→C;C→D(后代方向);
	// Z→A(更深祖先);V→W、W→V(双向环)。
	facts := []changeset.SupersededFact{
		mk("sha256:a", "sha256:b1", fixtureAt),
		mk("sha256:a", "sha256:b2", fixtureAt.Add(time.Minute)),
		mk("sha256:b1", "sha256:c", fixtureAt2),
		mk("sha256:b2", "sha256:c", fixtureAt2.Add(time.Minute)),
		mk("sha256:c", "sha256:d", fixtureAt3), // C 被 D 取代:从 C 走是后代方向
		mk("sha256:z", "sha256:a", fixtureAt3.Add(time.Minute)),
		mk("sha256:v", "sha256:w", fixtureAt),
		mk("sha256:w", "sha256:v", fixtureAt2), // 环:W↔V
	}
	graph := lineageFactGraph(t, facts...)
	reversed := func(id changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
		fs, err := graph(id)
		if err != nil {
			return nil, err
		}
		out := make([]changeset.SupersededFact, len(fs))
		for i, f := range fs {
			out[len(fs)-1-i] = f
		}
		return out, nil
	}

	cases := []struct {
		from string
		want []string
	}{
		// C 的祖先:第 1 层 {b1, b2}(身份排序),第 2 层 a,第 3 层 z;
		// C→D 是 C 的后代,不进入。
		{"sha256:c", []string{"sha256:b1", "sha256:b2", "sha256:a", "sha256:z"}},
		{"sha256:d", []string{"sha256:c", "sha256:b1", "sha256:b2", "sha256:a", "sha256:z"}},
		{"sha256:b1", []string{"sha256:a", "sha256:z"}},
		{"sha256:a", []string{"sha256:z"}},
		// 孤点(无取代事实):空祖先。
		{"sha256:lonely", nil},
		// 环:visited 截断,恰好一层后终止。
		{"sha256:w", []string{"sha256:v"}},
	}
	for _, c := range cases {
		got, err := walkSupersessionAncestors(changeset.ChangeSetID(c.from), graph)
		if err != nil {
			t.Fatalf("ancestors(%s): %v", c.from, err)
		}
		var gotIDs []string
		for _, id := range got {
			gotIDs = append(gotIDs, string(id))
		}
		if strings.Join(gotIDs, ",") != strings.Join(c.want, ",") {
			t.Fatalf("ancestors(%s) = %v, want %v", c.from, gotIDs, c.want)
		}
		// 确定性:事实返回顺序颠倒,输出不变。
		again, err := walkSupersessionAncestors(changeset.ChangeSetID(c.from), reversed)
		if err != nil {
			t.Fatalf("ancestors-reversed(%s): %v", c.from, err)
		}
		if !reflect.DeepEqual(got, again) {
			t.Fatalf("ancestors(%s) 非确定: %v != %v", c.from, got, again)
		}
	}

	// 非法输入:空身份拒绝(fail-safe,不静默当作「无祖先」)。
	if _, err := walkSupersessionAncestors("", graph); err == nil {
		t.Fatal("空 changeset id 应报错")
	}
	// facts 通道报错原样上抛,不被吞掉。
	if _, err := walkSupersessionAncestors("sha256:c", func(changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
		return nil, errors.New("boom")
	}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("读取错误应上抛,得到 %v", err)
	}
}

// TestExposuresForChangeSetLineageDB 固化 P0-07 AC1/AC3/AC4(数据库层):
// lineage 投影按「当前候选 + 取代祖先」展开,当前候选自身干净时祖先暴露
// 不消失;投影是纯读取,不产生任何可变暴露状态。
func TestExposuresForChangeSetLineageDB(t *testing.T) {
	st, pool := testStore(t)
	ctx := context.Background()
	project, task := core.ProjectID("proj"), "T-lineage"

	// 三代链:csA ← csB ← csC(csA 最老,csC 最新)。
	runA := startTestRun(t, ctx, st, project, task)
	csA := mustChangeSet(t, project, "base1", "", "a", testChangeSetFiles("a"), nil, runA)
	runB := startTestRun(t, ctx, st, project, task)
	csB := mustChangeSet(t, project, "base1", "", "b", testChangeSetFiles("b"), nil, runB)
	runC := startTestRun(t, ctx, st, project, task)
	csC := mustChangeSet(t, project, "base1", "", "c", testChangeSetFiles("c"), nil, runC)
	for _, cs := range []*changeset.ChangeSet{csA, csB, csC} {
		if err := st.SaveChangeSet(ctx, cs); err != nil {
			t.Fatalf("SaveChangeSet(%s): %v", cs.ID(), err)
		}
	}
	for _, pair := range []struct {
		old, new *changeset.ChangeSet
		at       time.Time
	}{
		{csA, csB, fixtureAt},
		{csB, csC, fixtureAt2},
	} {
		fact, err := changeset.NewSupersededFact(pair.old.ID(), pair.new.ID(), changeset.SupersedeRework, pair.at)
		if err != nil {
			t.Fatalf("NewSupersededFact: %v", err)
		}
		if err := st.AppendSupersededFact(ctx, fact); err != nil {
			t.Fatalf("AppendSupersededFact: %v", err)
		}
	}

	binding := evidence.BindingRef{Project: project, Verification: evidence.VerificationRevision(fixtureHash), BindingID: "b-held"}
	// 祖先 A 上的 held-out 暴露(lineage 局部通道)。
	eA, err := st.RecordExposure(ctx, ExposureRecord{
		Task: task, ContractRevision: 2, Binding: binding, Acceptance: "AC1",
		ChangeSet: evidence.ChangeSetID(csA.ID()),
		Channel:   evidence.ChannelDebugDisclosure, Reason: "ancestor leak", ExposedAt: fixtureAt,
	})
	if err != nil {
		t.Fatalf("RecordExposure(A): %v", err)
	}
	// 当前 C 上的暴露(持久通道)。
	eC, err := st.RecordExposure(ctx, ExposureRecord{
		Task: task, ContractRevision: 2, Binding: binding, Acceptance: "AC1",
		ChangeSet: evidence.ChangeSetID(csC.ID()),
		Channel:   evidence.ChannelKnowledgeIndex, Reason: "current", ExposedAt: fixtureAt3,
	})
	if err != nil {
		t.Fatalf("RecordExposure(C): %v", err)
	}
	// 未锚定暴露:没有 lineage 坐标,不属于 lineage 投影。
	if _, err := st.RecordExposure(ctx, ExposureRecord{
		Task: task, ContractRevision: 2, Binding: binding, Channel: evidence.ChannelManual, ExposedAt: fixtureAt3,
	}); err != nil {
		t.Fatalf("RecordExposure(unanchored): %v", err)
	}

	// AC1:祖先展开确定性 —— 两次调用同输出;C → [B, A](最近在前);B → [A];A → []。
	ancC, err := st.ChangeSetAncestors(ctx, csC.ID())
	if err != nil {
		t.Fatalf("ChangeSetAncestors(C): %v", err)
	}
	if len(ancC) != 2 || ancC[0] != csB.ID() || ancC[1] != csA.ID() {
		t.Fatalf("ancestors(C) = %v, want [B, A]", ancC)
	}
	ancC2, err := st.ChangeSetAncestors(ctx, csC.ID())
	if err != nil || !reflect.DeepEqual(ancC, ancC2) {
		t.Fatalf("ancestors(C) 非确定: %v != %v(%v)", ancC, ancC2, err)
	}
	ancB, err := st.ChangeSetAncestors(ctx, csB.ID())
	if err != nil || len(ancB) != 1 || ancB[0] != csA.ID() {
		t.Fatalf("ancestors(B) = %v(%v), want [A]", ancB, err)
	}
	ancA, err := st.ChangeSetAncestors(ctx, csA.ID())
	if err != nil || len(ancA) != 0 {
		t.Fatalf("ancestors(A) = %v(%v), want 空", ancA, err)
	}
	if _, err := st.ChangeSetAncestors(ctx, ""); err == nil {
		t.Fatal("空 changeset id 应报错")
	}

	// AC3:B 自身干净(候选局部投影为空),祖先暴露经 lineage 投影仍可见。
	forB, err := st.ExposuresForChangeSet(ctx, project, csB.ID())
	if err != nil || len(forB) != 0 {
		t.Fatalf("B 的候选局部暴露应空: %v %d", err, len(forB))
	}
	lineageB, err := st.ExposuresForChangeSetLineage(ctx, project, csB.ID())
	if err != nil {
		t.Fatalf("ExposuresForChangeSetLineage(B): %v", err)
	}
	if len(lineageB) != 1 || lineageB[0].Exposure.ID() != eA.ID() ||
		lineageB[0].ChangeSet != csA.ID() || lineageB[0].AncestorDepth != 1 {
		t.Fatalf("lineage(B) = %+v, want 仅祖先 A 上的暴露(depth 1)", lineageB)
	}

	// C:当前暴露在前(depth 0),祖先暴露随后(depth 2);顺序确定。
	lineageC, err := st.ExposuresForChangeSetLineage(ctx, project, csC.ID())
	if err != nil {
		t.Fatalf("ExposuresForChangeSetLineage(C): %v", err)
	}
	if len(lineageC) != 2 {
		t.Fatalf("lineage(C) 条数 = %d, want 2(未锚定暴露不计入)", len(lineageC))
	}
	if lineageC[0].Exposure.ID() != eC.ID() || lineageC[0].ChangeSet != csC.ID() || lineageC[0].AncestorDepth != 0 {
		t.Fatalf("lineage(C)[0] = %+v, want 当前候选暴露(depth 0)", lineageC[0])
	}
	if lineageC[1].Exposure.ID() != eA.ID() || lineageC[1].ChangeSet != csA.ID() || lineageC[1].AncestorDepth != 2 {
		t.Fatalf("lineage(C)[1] = %+v, want 祖先 A 暴露(depth 2)", lineageC[1])
	}
	lineageC2, err := st.ExposuresForChangeSetLineage(ctx, project, csC.ID())
	if err != nil || !reflect.DeepEqual(lineageC, lineageC2) {
		t.Fatalf("lineage(C) 非确定: %v != %v(%v)", lineageC, lineageC2, err)
	}

	// AC4:投影是纯读取 —— 不产生新暴露行、不改写既有行,事实面不存在
	// 任何暴露状态字段可变(API 面由 TestStoreAPIHasNoFactMutators 固化)。
	if n := countRows(t, pool, "SELECT count(*) FROM evidence_exposure"); n != 3 {
		t.Fatalf("lineage 投影后暴露行数 = %d, want 3(投影零写副作用)", n)
	}
	mustModifyRejected(t, pool, "UPDATE evidence_exposure SET changeset_id=$1 WHERE id=$2", string(csC.ID()), string(eA.ID()))
	if still, err := st.ExposuresForChangeSetLineage(ctx, project, csC.ID()); err != nil ||
		len(still) != 2 || still[1].Exposure.ID() != eA.ID() || still[1].ChangeSet != csA.ID() {
		t.Fatalf("拒绝改写后 lineage 投影不一致: %v %+v", err, still)
	}

	// 缺少 project 参数:适配器拒绝(fail-safe)。
	if _, err := st.ExposuresForChangeSetLineage(ctx, "", csC.ID()); err == nil {
		t.Fatal("空 project 应报错")
	}
}

func TestRunstorePackageBuilds(t *testing.T) {
	// 兜底哨兵:保证无 DB 环境下包与测试文件本身参与编译并通过基本断言。
	if fmt.Sprint(reflect.TypeOf(Store{}).Name()) != "Store" {
		t.Fatal("包类型缺失")
	}
	if _, err := encodeStrings([]string{"x"}); err != nil {
		t.Fatalf("encodeStrings: %v", err)
	}
}
