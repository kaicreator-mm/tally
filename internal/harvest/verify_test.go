package harvest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"tally/internal/core"
)

// fakeVerifier 是注入的假验证器:可配置通过/失败与返回的版本组合。
type fakeVerifier struct {
	err    error
	res    VerifyResult
	calls  int
	gotDef Definition
}

func (f *fakeVerifier) Verify(_ context.Context, def Definition) (VerifyResult, error) {
	f.calls++
	f.gotDef = def
	if f.err != nil {
		return VerifyResult{}, f.err
	}
	return f.res, nil
}

var (
	testCombo = core.VersionCombo{"docker": "27.0", "image:0": "alpine:latest"}
	fixedNow  = time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
)

func testDef() Definition {
	return Definition{
		Name:         "minio",
		Kind:         KindDockerCompose,
		Compose:      "services:\n  minio:\n    image: alpine\n",
		HealthChecks: []HealthCheck{{Name: "hc", Type: CheckLogPattern, Service: "minio", Pattern: "ready"}},
	}
}

// 验收:验证通过 → 证据有效,带验证时间与完整版本组合。
func TestVerifyPass(t *testing.T) {
	fv := &fakeVerifier{res: VerifyResult{VerifiedAt: fixedNow, Combo: testCombo, Record: "health=1/1ok"}}
	proof, err := Verify(context.Background(), fv, testDef(), nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !proof.Valid() {
		t.Error("通过验证的证据应有效")
	}
	if !proof.VerifiedAt().Equal(fixedNow) {
		t.Errorf("验证时刻应来自 Verifier: %v", proof.VerifiedAt())
	}
	if !reflect.DeepEqual(proof.VerifiedCombo(), testCombo) {
		t.Errorf("证据应带完整版本组合: %v", proof.VerifiedCombo())
	}
	if proof.GateRecord() != "health=1/1ok" {
		t.Errorf("应带机器记录: %q", proof.GateRecord())
	}
}

// VerifiedAt 零值时用注入时钟兜底(验证时刻总得有)。
func TestVerifyZeroTimeUsesNow(t *testing.T) {
	fv := &fakeVerifier{res: VerifyResult{Combo: testCombo}}
	proof, err := Verify(context.Background(), fv, testDef(), func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !proof.VerifiedAt().Equal(fixedNow) {
		t.Errorf("VerifiedAt 零值应回退注入时钟: %v", proof.VerifiedAt())
	}
}

// 验收:验证失败 → 证据零值、无效,错误上抛。
func TestVerifyFail(t *testing.T) {
	fv := &fakeVerifier{err: errors.New("健康检查不过")}
	proof, err := Verify(context.Background(), fv, testDef(), nil)
	if err == nil {
		t.Fatal("验证失败应返回错误")
	}
	if proof.Valid() {
		t.Error("失败的证据应无效")
	}
}

// 验收(铁律 2 编译期基础):证据类型字段全部不导出(仿 failure 的反射断言)。
func TestProofFieldsUnexported(t *testing.T) {
	rt := reflect.TypeOf(Proof{})
	if rt.NumField() == 0 {
		t.Fatal("证据类型应携带验证时刻/版本组合/记录")
	}
	for i := 0; i < rt.NumField(); i++ {
		if f := rt.Field(i); f.IsExported() {
			t.Errorf("证据字段 %s 导出了:包外可伪造证据,违反铁律 2", f.Name)
		}
	}
}

// 零值证据无效。
func TestProofZeroInvalid(t *testing.T) {
	if (Proof{}).Valid() {
		t.Error("零值证据不应有效")
	}
}

// Proof 实现 core.VerificationProof(编译期)。
func TestProofImplementsCoreProof(t *testing.T) {
	var p core.VerificationProof = Proof{}
	if p.Valid() {
		t.Error("零值证据不应有效")
	}
}
