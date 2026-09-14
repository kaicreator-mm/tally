package harvest

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v3"

	"tally/internal/core"
	"tally/internal/gitwriter"
)

// initCatalogRepo 在临时目录初始化一个 git 仓库。
func initCatalogRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

func fixedIdentity() Identity {
	return Identity{Name: "tally-test", Email: "test@example.com"}
}

// proofVia 走真实的 Verify 路径(fakeVerifier)得到带内容的证据,保证证据路径一致。
func proofVia(t *testing.T, combo core.VersionCombo) Proof {
	t.Helper()
	fv := &fakeVerifier{res: VerifyResult{VerifiedAt: fixedNow, Combo: combo, Record: "health=1/1ok"}}
	p, err := Verify(context.Background(), fv, testDef(), func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("造证据: %v", err)
	}
	return p
}

// 验收:只有验证通过(Proof 有效)才写 catalog;写三个文件 + git commit。
func TestStoreWritesCatalogAndCommits(t *testing.T) {
	repo := initCatalogRepo(t)
	c := &GitCatalog{RepoPath: repo, Identity: fixedIdentity()}

	def := testDef()
	def.Source = "coolify:minio"
	proof := proofVia(t, testCombo)

	if err := c.Store(context.Background(), def, proof); err != nil {
		t.Fatalf("Store: %v", err)
	}

	dir := filepath.Join(repo, "catalog", "minio")
	for _, f := range []string{"definition.yaml", "compose.yaml", "verified.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("应写入 %s: %v", f, err)
		}
	}

	// verified.json 含验证时间 + 完整版本组合 + 出处。
	b, err := os.ReadFile(filepath.Join(dir, "verified.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec verifiedRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatalf("verified.json 应是合法 JSON: %v", err)
	}
	if rec.VerifiedAt.IsZero() {
		t.Error("verified.json 应含验证时间")
	}
	if !reflect.DeepEqual(rec.Combo, testCombo) {
		t.Errorf("verified.json 应含完整版本组合: %v", rec.Combo)
	}
	if rec.Source != "coolify:minio" {
		t.Errorf("verified.json 应含出处: %q", rec.Source)
	}

	// definition.yaml 里 compose_file 指向同目录 compose.yaml。
	defBytes, _ := os.ReadFile(filepath.Join(dir, "definition.yaml"))
	var back Definition
	if err := yaml.Unmarshal(defBytes, &back); err != nil {
		t.Fatalf("definition.yaml 应可解析: %v", err)
	}
	if back.ComposeFile != "compose.yaml" {
		t.Errorf("definition.yaml 的 compose_file 应指向 compose.yaml: %q", back.ComposeFile)
	}

	// git 有提交。
	r, _ := git.PlainOpen(repo)
	head, err := r.Head()
	if err != nil || head.Hash().IsZero() {
		t.Errorf("Store 应产生 git 提交: %v", err)
	}
}

// 验收(铁律 2):写入函数拒绝无效证据(零值 Proof),且不写任何文件。
func TestStoreRejectsInvalidProof(t *testing.T) {
	repo := initCatalogRepo(t)
	c := &GitCatalog{RepoPath: repo, Identity: fixedIdentity()}
	err := c.Store(context.Background(), testDef(), Proof{}) // 零值证据
	if err == nil {
		t.Fatal("无效证据不得写入 catalog")
	}
	if _, statErr := os.Stat(filepath.Join(repo, "catalog", "minio")); !os.IsNotExist(statErr) {
		t.Error("无效证据不应写任何文件")
	}
}

// 写入缺 name 的定义 → 拒绝。
func TestStoreRejectsNameless(t *testing.T) {
	repo := initCatalogRepo(t)
	c := &GitCatalog{RepoPath: repo, Identity: fixedIdentity()}
	def := testDef()
	def.Name = ""
	if err := c.Store(context.Background(), def, proofVia(t, testCombo)); err == nil {
		t.Fatal("缺 name 的定义不得写入")
	}
}

// 验收(单一写边界):提交经 internal/gitwriter——内容与 HEAD 完全一致时,
// gitwriter 以 gitwriter.ErrNoChanges 拒绝且不产生重复提交(旧 go-git 直写路径
// 会无声再造一个同树提交)。这个类型化错误只有 gitwriter 路径能给出。
func TestStoreNoChangesIsGitWriterTypedError(t *testing.T) {
	repo := initCatalogRepo(t)
	c := &GitCatalog{RepoPath: repo, Identity: fixedIdentity()}
	def := testDef()
	proof := proofVia(t, testCombo)

	if err := c.Store(context.Background(), def, proof); err != nil {
		t.Fatalf("首次 Store: %v", err)
	}
	r, _ := git.PlainOpen(repo)
	first, err := r.Head()
	if err != nil {
		t.Fatalf("首次 Store 后应提交: %v", err)
	}

	err = c.Store(context.Background(), def, proof)
	if !errors.Is(err, gitwriter.ErrNoChanges) {
		t.Fatalf("重复收录应报 gitwriter.ErrNoChanges: %v", err)
	}
	second, err := r.Head()
	if err != nil {
		t.Fatalf("读 HEAD: %v", err)
	}
	if second.Hash() != first.Hash() {
		t.Error("无变化不得产生重复提交")
	}
}
