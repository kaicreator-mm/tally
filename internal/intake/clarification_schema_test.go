package intake

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tally/internal/airuntime"
)

// proposeClarification 用 FakeExecutor 跑一次纯澄清提案,返回解析结果。
func proposeClarification(t *testing.T, out string) (ClarificationProposal, error) {
	t.Helper()
	fake := &airuntime.FakeExecutor{Result: succeededResult(out)}
	p, err := Propose(context.Background(), fake, Intent{Text: "登录有问题,修一下"})
	if p == nil {
		return ClarificationProposal{}, err
	}
	cp, ok := p.(ClarificationProposal)
	if !ok {
		t.Fatalf("应返回 ClarificationProposal, got %T", p)
	}
	return cp, err
}

// AC1(兼容):旧简单候选澄清不带 kind → 显式迁移为 choice,问题与候选
// 原样保留;旧开放问题(无候选)→ 迁移为 free_text(自然语言逃生舱)。
func TestClarificationLegacyShapeMigratesCompatibly(t *testing.T) {
	cp, err := proposeClarification(t,
		`{"clarifications":[{"question":"缺陷在哪个登录入口?","candidates":["Web","移动端"]},{"question":"复现步骤是什么?"}]}`)
	if err != nil {
		t.Fatalf("旧形态应原样兼容, got %v", err)
	}
	if cp.Questions[0].Kind != KindChoice {
		t.Errorf("带候选的旧形态应迁移为 choice, got %q", cp.Questions[0].Kind)
	}
	if cp.Questions[0].Question != "缺陷在哪个登录入口?" || len(cp.Questions[0].Candidates) != 2 {
		t.Errorf("旧形态字段应原样保留: %+v", cp.Questions[0])
	}
	if cp.Questions[1].Kind != KindFreeText {
		t.Errorf("无候选的旧开放问题应迁移为 free_text, got %q", cp.Questions[1].Kind)
	}
}

// 正例:有界词表内的类型 + advisory 建议照常解析;OutputShape 向推理声明
// 扩展后的严格输出形状(含 kind / recommended / recommendation_reason)。
func TestClarificationBoundedKindsAccepted(t *testing.T) {
	cp, err := proposeClarification(t, `{"clarifications":[
		{"question":"兼容策略选哪个?","kind":"risk_choice","candidates":["保持现有 API","允许破坏性变更"],
		 "recommended":0,"recommendation_reason":"现有调用方多,破坏性变更风险高"},
		{"question":"是否保留旧端点?","kind":"boolean"},
		{"question":"保留多少天?","kind":"number","min":1,"max":90},
		{"question":"还有什么补充?","kind":"free_text"}
	]}`)
	if err != nil {
		t.Fatalf("有界词表内的类型应接受: %v", err)
	}
	if len(cp.Questions) != 4 {
		t.Fatalf("问题数 = %d, want 4", len(cp.Questions))
	}
	rc := cp.Questions[0]
	if rc.Kind != KindRiskChoice || rc.Recommended == nil || *rc.Recommended != 0 {
		t.Errorf("risk_choice + advisory 推荐应照常携带: %+v", rc)
	}
	if rc.RecommendationReason == "" {
		t.Errorf("推荐理由应作为数据携带: %+v", rc)
	}
	if cp.Questions[1].Kind != KindBoolean || cp.Questions[2].Kind != KindNumber || cp.Questions[3].Kind != KindFreeText {
		t.Errorf("类型词表应原样收敛: %+v", cp.Questions)
	}

	fake := &airuntime.FakeExecutor{Result: succeededResult(`{"clarifications":[{"question":"q","kind":"choice","candidates":["a"]}]}`)}
	if _, err := Propose(context.Background(), fake, Intent{Text: "修一下"}); err != nil {
		t.Fatalf("Propose 出错: %v", err)
	}
	req := fake.Requests()[0]
	// 封闭词表与 advisory 语义在提示词里向推理显式声明(OutputShape 仍是
	// 顶层形状样例,既有行为不变)。
	for _, want := range []string{"choice/multichoice/boolean/number", "free_text", "advisory"} {
		if !strings.Contains(req.Input.Prompt, want) {
			t.Errorf("提示词应声明有界词表 %q: %q", want, req.Input.Prompt)
		}
	}
	if !jsonValid(req.Input.OutputShape) || !strings.Contains(string(req.Input.OutputShape), "clarifications") {
		t.Errorf("OutputShape 应仍是含 clarifications 的合法 JSON: %s", req.Input.OutputShape)
	}
}

// jsonValid 报告 raw 是否为合法 JSON(测试内小助手,不引第三方)。
func jsonValid(raw []byte) bool {
	var v any
	return json.Unmarshal(raw, &v) == nil
}

// AC2(负例,DL-11 fail safe):词表之外的类型一律拒绝,绝不猜测成已知类型;
// 词表是精确匹配,大小写漂移同样拒绝。
func TestClarificationUnknownKindRejected(t *testing.T) {
	for _, kind := range []string{"dropdown", "html", "CHOICE", "custom_widget", "choice "} {
		_, err := proposeClarification(t,
			`{"clarifications":[{"question":"q","kind":"`+kind+`","candidates":["a"]}]}`)
		if !errors.Is(err, ErrUnknownClarificationKind) {
			t.Errorf("kind=%q 应 ErrUnknownClarificationKind, got %v", kind, err)
		}
	}
}

// 负例(fail safe):类型与形状不匹配、数量/长度超限、advisory 越界 ——
// 一律 ErrInvalidProposal,不静默截断、不猜测。未知 JSON 字段仍被严格解析
// 拒绝(严格结构化输出不放宽)。
func TestClarificationShapeViolationsRejected(t *testing.T) {
	cases := []struct {
		name string
		out  string
	}{
		{"选项类缺候选", `{"clarifications":[{"question":"q","kind":"choice"}]}`},
		{"boolean 不得带候选", `{"clarifications":[{"question":"q","kind":"boolean","candidates":["是"]}]}`},
		{"free_text 不得带候选", `{"clarifications":[{"question":"q","kind":"free_text","candidates":["a"]}]}`},
		{"recommended 越界", `{"clarifications":[{"question":"q","kind":"choice","candidates":["a"],"recommended":1}]}`},
		{"recommended 负数", `{"clarifications":[{"question":"q","kind":"choice","candidates":["a"],"recommended":-1}]}`},
		{"boolean 不支持 recommended", `{"clarifications":[{"question":"q","kind":"boolean","recommended":0}]}`},
		{"推荐理由必须伴随 recommended", `{"clarifications":[{"question":"q","kind":"choice","candidates":["a"],"recommendation_reason":"因为"}]}`},
		{"数值区间倒挂", `{"clarifications":[{"question":"q","kind":"number","min":10,"max":1}]}`},
		{"问题超长", `{"clarifications":[{"question":"` + strings.Repeat("问", maxClarificationQuestionLen+1) + `"}]}`},
		{"候选标签超长", `{"clarifications":[{"question":"q","kind":"choice","candidates":["` + strings.Repeat("a", maxClarificationOptionLen+1) + `"]}]}`},
		{"推荐理由超长", `{"clarifications":[{"question":"q","kind":"choice","candidates":["a"],"recommended":0,"recommendation_reason":"` + strings.Repeat("理", maxClarificationReasonLen+1) + `"}]}`},
		{"未知 JSON 字段仍拒绝", `{"clarifications":[{"question":"q","kind":"choice","candidates":["a"],"color":"red"}]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := proposeClarification(t, c.out); !errors.Is(err, ErrInvalidProposal) {
				t.Fatalf("应 ErrInvalidProposal, got %v", err)
			}
		})
	}

	// 问题数超上限(上限+1 个合法问题)。
	qs := make([]string, 0, maxClarifications+1)
	for i := 0; i <= maxClarifications; i++ {
		qs = append(qs, `{"question":"q`+strings.Repeat("i", i)+`","kind":"free_text"}`)
	}
	if _, err := proposeClarification(t, `{"clarifications":[`+strings.Join(qs, ",")+`]}`); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("问题数超上限应 ErrInvalidProposal, got %v", err)
	}

	// 候选数超上限(上限+1 个合法候选)。
	cs := make([]string, 0, maxClarificationOptions+1)
	for i := 0; i <= maxClarificationOptions; i++ {
		cs = append(cs, `"c`+strings.Repeat("i", i)+`"`)
	}
	if _, err := proposeClarification(t, `{"clarifications":[{"question":"q","kind":"multichoice","candidates":[`+strings.Join(cs, ",")+`]}]}`); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("候选数超上限应 ErrInvalidProposal, got %v", err)
	}
}

// 验证(合同:injection-shaped values rendered as data):注入形态的问题/
// 候选值原样作为纯文本数据携带 —— schema 没有任何标记/脚本承载面,不做
// HTML 执行,也不静默改写;渲染层(html/template)按数据转义,权威语义
// 不受展示值影响(DL-11 / PRD §8.4)。
func TestClarificationInjectionValuesStayData(t *testing.T) {
	payloads := []string{
		`<script>alert("xss")</script>`,
		`"><img src=x onerror=alert(1)>`,
		`'; DROP TABLE users; --`,
		`<a href="javascript:alert(1)">点我</a>`,
	}
	for i, p := range payloads {
		cp, err := proposeClarification(t,
			`{"clarifications":[{"question":"`+escJSON(p)+`","kind":"choice","candidates":["`+escJSON(p)+`"]}]}`)
		if err != nil {
			t.Fatalf("注入形态值是合法数据,应被 schema 接受(payload #%d): %v", i, err)
		}
		q := cp.Questions[0]
		if q.Question != p || q.Candidates[0] != p {
			t.Errorf("payload #%d 应原样作为数据携带,不执行也不改写:\n got q=%q cand=%q\nwant %q", i, q.Question, q.Candidates[0], p)
		}
		if q.Kind != KindChoice {
			t.Errorf("payload #%d 应显式迁移为 choice, got %q", i, q.Kind)
		}
	}
}

// escJSON 把字符串转成 JSON 字符串字面量(不含引号包裹),供测试拼 wire 输出。
func escJSON(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '<':
			b.WriteString(`\u003c`)
		case '>':
			b.WriteString(`\u003e`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
