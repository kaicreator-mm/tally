package verification_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tally/internal/contextpack"
	"tally/internal/holdout"
	"tally/internal/verification"
)

// TestHoldoutPathLeakInContextPackProjection 最初是 P1-27 的 holdout 端到端 AC4 证明。
// P1-29 起本包(verification)生产导入 holdout,该测试若留在 holdout 包内会形成
// 测试导入环,故按集成者修正迁到本包(两个被测面的断言语义原样保留)。
func TestHoldoutPathLeakInContextPackProjection(t *testing.T) {
	project := t.TempDir()
	root := filepath.Join(t.TempDir(), "holdout")
	s, err := holdout.NewFilesystemStore(root, project)
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	ctx := context.Background()
	ref, err := s.Put(ctx, []byte("hidden failure scenario"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// 冻结一份含 held_out binding(store/ref 指向该 store)的 VerificationContract,
	// 再推导实现权威可见的 redacted projection(§8.4.3)。
	contract := &verification.Contract{
		TaskID:   "T-P1-27",
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:        "AC-1",
			Statement: "hidden validator detects the seeded defect",
			EvidenceBindings: []verification.Binding{{
				ID:               "b-hidden",
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityHeldOut,
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: ref.String()},
			}},
		}},
		RuntimeRequirements: verification.RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
	}
	frozen, err := contract.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	projections, err := contextpack.RedactVerification(frozen)
	if err != nil {
		t.Fatalf("RedactVerification: %v", err)
	}
	if len(projections) != 1 || len(projections[0].Evidence) != 1 {
		t.Fatalf("投影形状不符: %+v", projections)
	}
	if projections[0].Evidence[0].Visibility != verification.VisibilityHeldOut {
		t.Fatalf("可见性 = %q, want held_out", projections[0].Evidence[0].Visibility)
	}
	if projections[0].Evidence[0].Store != "" || projections[0].Evidence[0].Ref != "" {
		t.Fatalf("held_out 投影携带 store/ref: %+v", projections[0].Evidence[0])
	}

	rendered := fmt.Sprintf("%+v", projections)
	jsonRaw, err := json.Marshal(projections)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	jsonRendered := string(jsonRaw)
	leaks := []string{root, filepath.Join(root, "objects"), project, ref.String(), "b-hidden", "holdout"}
	if runtime.GOOS == "windows" {
		// JSON 转义的反斜杠形态也必须不出现。
		leaks = append(leaks, strings.ReplaceAll(root, `\`, `\`))
	}
	for _, leak := range leaks {
		if strings.Contains(rendered, leak) {
			t.Fatalf("ContextPack 投影泄漏 %q: %s", leak, rendered)
		}
		if strings.Contains(jsonRendered, leak) {
			t.Fatalf("ContextPack 投影 JSON 泄漏 %q: %s", leak, jsonRendered)
		}
	}
}
