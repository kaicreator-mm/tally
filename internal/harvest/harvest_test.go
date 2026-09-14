package harvest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeCatalog 记录哪些服务被写入(断言「只有验证通过的才写」)。
type fakeCatalog struct {
	stored []string
	err    error
}

func (f *fakeCatalog) Store(_ context.Context, def Definition, _ Proof) error {
	if f.err != nil {
		return f.err
	}
	f.stored = append(f.stored, def.Name)
	return nil
}

// selectiveVerifier 按服务名决定成败。
type selectiveVerifier struct {
	pass  map[string]bool
	calls int
}

func (s *selectiveVerifier) Verify(_ context.Context, def Definition) (VerifyResult, error) {
	s.calls++
	if s.pass[def.Name] {
		return VerifyResult{VerifiedAt: fixedNow, Combo: testCombo, Record: "health=1/1ok"}, nil
	}
	return VerifyResult{}, errors.New("起不来")
}

func newHarvester(fe *fakeExecutor, fv Verifier, fc *fakeCatalog, src Source) *Harvester {
	return &Harvester{
		Executor: fe,
		Verifier: fv,
		Catalog:  fc,
		Sources:  map[string]Source{src.Name(): src},
		Now:      func() time.Time { return fixedNow },
	}
}

func staticSrc(content map[string]string) StaticSource {
	return StaticSource{SourceName: "coolify", Content: content}
}

// 验收:验证通过 → 写入 catalog(Stored)。
func TestRunStoresVerified(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	fv := &fakeVerifier{res: VerifyResult{VerifiedAt: fixedNow, Combo: testCombo, Record: "ok"}}
	fc := &fakeCatalog{}
	h := newHarvester(fe, fv, fc, staticSrc(map[string]string{"minio": "raw"}))

	out := h.Run(context.Background(), []Service{{Name: "minio", Source: "coolify", Ref: "minio"}})
	if len(out) != 1 {
		t.Fatalf("应有一个结果, got %d", len(out))
	}
	if !out[0].Stored || out[0].Skipped {
		t.Errorf("验证通过应 Stored: %+v", out[0])
	}
	if len(fc.stored) != 1 || fc.stored[0] != "minio" {
		t.Errorf("应写入 minio: %v", fc.stored)
	}
}

// 验收:只有验证通过的写 catalog;未验证的不写,且跳过带原因。
func TestRunOnlyVerifiedWritten(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	fv := &selectiveVerifier{pass: map[string]bool{"minio": true}} // minio 过,redis 挂
	fc := &fakeCatalog{}
	h := newHarvester(fe, fv, fc, staticSrc(map[string]string{"minio": "raw", "redis": "raw"}))

	out := h.Run(context.Background(), []Service{
		{Name: "minio", Source: "coolify", Ref: "minio"},
		{Name: "redis", Source: "coolify", Ref: "redis"},
	})

	if len(fc.stored) != 1 || fc.stored[0] != "minio" {
		t.Errorf("只有验证通过的 minio 应写入, got %v", fc.stored)
	}
	var redis Outcome
	for _, o := range out {
		if o.Service == "redis" {
			redis = o
		}
	}
	if !redis.Skipped || redis.Reason == "" {
		t.Errorf("未验证的 redis 应跳过并记因: %+v", redis)
	}
}

// 验收:三轮不通过 → 跳过并记原因,第 4 次不再调 Verifier(不无限重试)。
func TestRunThreeRoundsThenSkip(t *testing.T) {
	fe := newFakeExecutor(goodProposal) // 提案总成功(含健康检查)
	fv := &fakeVerifier{err: errors.New("健康检查一直不过")}
	fc := &fakeCatalog{}
	h := newHarvester(fe, fv, fc, staticSrc(map[string]string{"minio": "raw"}))

	out := h.Run(context.Background(), []Service{{Name: "minio", Source: "coolify", Ref: "minio"}})

	if fv.calls != maxRounds {
		t.Errorf("应恰好在 %d 轮后停止调用 Verifier(第 4 次不调), got %d 次", maxRounds, fv.calls)
	}
	if len(fc.stored) != 0 {
		t.Errorf("三轮不过不应写入: %v", fc.stored)
	}
	if !out[0].Skipped || !strings.Contains(out[0].Reason, "跳过") {
		t.Errorf("应跳过并记因: %+v", out[0])
	}
}

// 提案一直失败(缺健康检查)→ 不进验证、三轮后跳过,Verifier 一次都不调。
func TestRunProposalFailureNeverVerifies(t *testing.T) {
	noHC := `{"summary":"x","compose":"services: {}","health_checks":[]}`
	fe := newFakeExecutor(noHC)
	fv := &fakeVerifier{}
	fc := &fakeCatalog{}
	h := newHarvester(fe, fv, fc, staticSrc(map[string]string{"minio": "raw"}))

	out := h.Run(context.Background(), []Service{{Name: "minio", Source: "coolify", Ref: "minio"}})

	if fv.calls != 0 {
		t.Errorf("提案失败不进验证,Verifier 不该被调, got %d", fv.calls)
	}
	if fe.calls() != maxRounds {
		t.Errorf("提案应尝试 %d 轮, got %d", maxRounds, fe.calls())
	}
	if !out[0].Skipped || len(fc.stored) != 0 {
		t.Errorf("应跳过且不写入: %+v", out[0])
	}
}

// 来源缺失 → 跳过记因。
func TestRunMissingSource(t *testing.T) {
	h := newHarvester(newFakeExecutor("{}"), &fakeVerifier{}, &fakeCatalog{}, staticSrc(map[string]string{}))
	out := h.Run(context.Background(), []Service{{Name: "x", Source: "unknown", Ref: "x"}})
	if !out[0].Skipped || !strings.Contains(out[0].Reason, "无来源适配") {
		t.Errorf("未知来源应跳过记因: %+v", out[0])
	}
}

// 写 catalog 失败 → 跳过记因(不算 Stored)。
func TestRunStoreErrorSkips(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	fv := &fakeVerifier{res: VerifyResult{VerifiedAt: fixedNow, Combo: testCombo}}
	fc := &fakeCatalog{err: errors.New("git 坏了")}
	h := newHarvester(fe, fv, fc, staticSrc(map[string]string{"minio": "raw"}))

	out := h.Run(context.Background(), []Service{{Name: "minio", Source: "coolify", Ref: "minio"}})
	if !out[0].Skipped || !strings.Contains(out[0].Reason, "写入 catalog 失败") {
		t.Errorf("写失败应跳过记因: %+v", out[0])
	}
}

// MaxRounds 可配置(覆盖默认 3)。
func TestRunMaxRoundsConfigurable(t *testing.T) {
	fe := newFakeExecutor(goodProposal)
	fv := &fakeVerifier{err: errors.New("不过")}
	fc := &fakeCatalog{}
	h := newHarvester(fe, fv, fc, staticSrc(map[string]string{"minio": "raw"}))
	h.MaxRounds = 5

	h.Run(context.Background(), []Service{{Name: "minio", Source: "coolify", Ref: "minio"}})
	if fv.calls != 5 {
		t.Errorf("MaxRounds=5 应调 5 次, got %d", fv.calls)
	}
}
