package requirement

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"

	"tally/internal/airuntime"
)

// succeededResult 构造一个 SUCCEEDED 的 Runtime 结果(airuntime.FakeExecutor 预设)。
func succeededResult(out string) airuntime.Result {
	return airuntime.Result{
		OperationID:       "op-test",
		Status:            airuntime.StatusSucceeded,
		TerminationReason: airuntime.TerminationCompleted,
		Output:            json.RawMessage(out),
	}
}

// 正例:推理把缺口转成带候选答案的问题;送出的 Request 过域侧闸且形状完整
// (operation/intent/幂等键/输出形状,DECISION_LOCKS #17 / PRD §28)。
func TestInterview(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(
		`{"questions":[{"gap":"REQ-1 磁盘上限","question":"草稿写满磁盘时怎么办?","candidates":["拒绝并提示","覆盖最旧草稿"]}]}`)}
	qs, err := Interview(context.Background(), fake, cleanDoc())
	if err != nil {
		t.Fatalf("Interview 出错: %v", err)
	}
	if len(qs) != 1 {
		t.Fatalf("问题数 = %d", len(qs))
	}
	if len(qs[0].Candidates) == 0 {
		t.Error("问题应尽量带候选答案")
	}

	reqs := fake.Requests()
	if len(reqs) != 1 {
		t.Fatalf("应恰好发出一次推理调用, got %d", len(reqs))
	}
	req := reqs[0]
	if req.Operation != InterviewOperation {
		t.Errorf("operation 应为稳定 ID %q, got %q", InterviewOperation, req.Operation)
	}
	if req.Intent.Role == "" || req.Intent.Instructions == "" {
		t.Errorf("intent 必须显式声明角色与用途: %+v", req.Intent)
	}
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 200 {
		t.Errorf("幂等键非法: %q(len=%d)", req.IdempotencyKey, len(req.IdempotencyKey))
	}
	if err := req.Validate(); err != nil {
		t.Errorf("域构造的 Request 应过域侧闸: %v", err)
	}
	if !strings.Contains(req.Input.Prompt, "REQ-1") {
		t.Errorf("提示词应含条款: %q", req.Input.Prompt)
	}
	if !strings.Contains(req.Input.Prompt, "不要重写") {
		t.Error("提示词应明确禁止重写正文")
	}
	if !json.Valid(req.Input.OutputShape) {
		t.Errorf("OutputShape 应是合法 JSON: %s", req.Input.OutputShape)
	}
}

// 负例:传输层失败(error 返回,无 Operation 结果)时 Interview 返回错误。
func TestInterviewExecutorError(t *testing.T) {
	fake := &airuntime.FakeExecutor{Err: errors.New("网络不可达")}
	if _, err := Interview(context.Background(), fake, cleanDoc()); err == nil {
		t.Fatal("传输层失败应返回错误")
	}
}

// 负例:UNCERTAIN/CANCELLED/FAILED 各归到可区分的类型化错误,不折叠成
// 一个"失败"(信任不变量:FAIL 与 ERROR/拿不准端到端可区分)。
func TestInterviewStatusMapping(t *testing.T) {
	cases := []struct {
		name   string
		status airuntime.Status
		want   error
	}{
		{"uncertain", airuntime.StatusUncertain, ErrInterviewUncertain},
		{"cancelled", airuntime.StatusCancelled, ErrInterviewCancelled},
		{"failed", airuntime.StatusFailed, ErrInterviewFailed},
	}
	all := []error{ErrInterviewUncertain, ErrInterviewCancelled, ErrInterviewFailed}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &airuntime.FakeExecutor{Result: airuntime.Result{
				OperationID:       "op-1",
				Status:            c.status,
				TerminationReason: airuntime.TerminationProviderFailure,
			}}
			_, err := Interview(context.Background(), fake, cleanDoc())
			if !errors.Is(err, c.want) {
				t.Fatalf("status=%s 应归 %v, got %v", c.status, c.want, err)
			}
			for _, other := range all {
				if other != c.want && errors.Is(err, other) {
					t.Errorf("status=%s 误命中 %v:终态必须可区分", c.status, other)
				}
			}
		})
	}
}

// 负例:SUCCEEDED 但输出为空 → 明确报错,不返回空问题清单假装成功。
func TestInterviewEmptyOutput(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: airuntime.Result{Status: airuntime.StatusSucceeded}}
	if _, err := Interview(context.Background(), fake, cleanDoc()); err == nil {
		t.Fatal("SUCCEEDED 但无输出应报错")
	}
}

// 负例:输出不是合法 JSON → 解析失败(域侧语义校验留在调用方,行为不变)。
func TestInterviewInvalidOutput(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult("不是 JSON")}
	if _, err := Interview(context.Background(), fake, cleanDoc()); err == nil {
		t.Fatal("非法 JSON 输出应解析失败")
	}
}

// 幂等键稳定性(契约 Idempotency-Key):同一文档重复问询复用同一键;
// 条款一变,键随之变(新调用)。
func TestInterviewIdempotencyKeyStable(t *testing.T) {
	fake := &airuntime.FakeExecutor{Result: succeededResult(`{"questions":[]}`)}
	doc := cleanDoc()
	for i := 0; i < 2; i++ {
		if _, err := Interview(context.Background(), fake, doc); err != nil {
			t.Fatalf("Interview 第 %d 次: %v", i+1, err)
		}
	}
	keys := []string{fake.Requests()[0].IdempotencyKey, fake.Requests()[1].IdempotencyKey}
	if keys[0] == "" || keys[0] != keys[1] {
		t.Errorf("同一文档应复用同一幂等键, got %q vs %q", keys[0], keys[1])
	}

	changed := cleanDoc()
	changed.Clauses[0].Statement += "(补充:上限 10MB)"
	if _, err := Interview(context.Background(), fake, changed); err != nil {
		t.Fatalf("Interview 条款变化后: %v", err)
	}
	if got := fake.Requests()[2].IdempotencyKey; got == keys[0] {
		t.Errorf("条款变化应产生新幂等键, got %q", got)
	}
}

// 脱敏闸:条款里混进疑似密钥 → 域侧 Request.Validate 拒绝(airuntime.ErrRedacted),
// 内容不到达传输(FakeExecutor 零调用)。
func TestInterviewRedactedPromptRejected(t *testing.T) {
	doc := cleanDoc()
	doc.Clauses[0].Statement = "WHEN 配置加载, 系统 shall 使用密钥 sk-abcdefghijklmnopqrst 认证"
	fake := &airuntime.FakeExecutor{Result: succeededResult(`{"questions":[]}`)}
	_, err := Interview(context.Background(), fake, doc)
	if !errors.Is(err, airuntime.ErrRedacted) {
		t.Fatalf("疑似密钥应被域侧脱敏闸拒绝(airuntime.ErrRedacted), got %v", err)
	}
	if n := len(fake.Requests()); n != 0 {
		t.Errorf("被拒内容不得到达传输, got %d 次调用", n)
	}
}

// 验收标准:送进 LLM 的输出结构里没有"需求正文"字段 —— LLM 不产出需求正文。
func TestProposalSchemaHasNoRequirementText(t *testing.T) {
	forbidden := map[string]bool{
		"text": true, "requirement": true, "body": true, "content": true,
		"spec": true, "draft": true, "正文": true, "需求正文": true, "requirement_text": true,
	}
	var check func(rt reflect.Type)
	check = func(rt reflect.Type) {
		switch rt.Kind() {
		case reflect.Slice, reflect.Ptr, reflect.Array:
			check(rt.Elem())
			return
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				name := strings.ToLower(strings.Split(f.Tag.Get("json"), ",")[0])
				if forbidden[name] {
					t.Errorf("提案结构出现需求正文字段 %q:LLM 不得产出需求正文", f.Name)
				}
				check(f.Type)
			}
		}
	}
	check(reflect.TypeOf(proposal{}))
}

// CanSign 只看结构缺口:实质缺口(问询建议)不阻塞签署。
func TestCanSign(t *testing.T) {
	if !CanSign(cleanDoc()) {
		t.Error("干净文档应可签署")
	}
	d := cleanDoc()
	d.NonGoals = nil // 造结构缺口(L4)
	if CanSign(d) {
		t.Error("有结构缺口时不可签署")
	}
}

// 驳回留痕:写进 git(rejections.md + 提交),追加式,不阻塞签署。
func TestRejectLeavesTraceInGit(t *testing.T) {
	repo := initRepo(t)
	q := Question{Gap: "REQ-1 磁盘上限", Question: "写满磁盘怎么办?", Candidates: []string{"拒绝", "覆盖"}}

	if _, err := Reject(repo, q, "本阶段不考虑磁盘上限", testIdentity()); err != nil {
		t.Fatalf("Reject 出错: %v", err)
	}
	// 再驳回一次 → 追加
	if _, err := Reject(repo, q, "再次确认不做", testIdentity()); err != nil {
		t.Fatalf("第二次 Reject 出错: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(repo, ".tally", "rejections.md"))
	if err != nil {
		t.Fatalf("驳回记录未写入: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "REQ-1 磁盘上限") || !strings.Contains(s, "本阶段不考虑磁盘上限") {
		t.Errorf("驳回记录缺内容: %q", s)
	}
	if strings.Count(s, "## 驳回") != 2 {
		t.Errorf("两次驳回应追加为两条记录, got %d", strings.Count(s, "## 驳回"))
	}

	// 留痕进了 git(有提交)
	r, _ := git.PlainOpen(repo)
	head, err := r.Head()
	if err != nil || head.Hash().IsZero() {
		t.Errorf("驳回应产生 git 提交: %v", err)
	}
}
