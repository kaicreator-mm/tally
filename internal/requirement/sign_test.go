package requirement

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"

	"tally/internal/gitwriter"
)

// fakeSigner 是注入的假签名器:实现 go-git 的 git.Signer,返回固定签名。
// 测试不依赖真实 GPG,但签名路径真实发生(提交对象真的带上了签名)。
type fakeSigner struct {
	sig    []byte
	called bool
}

func (f *fakeSigner) Sign(r io.Reader) ([]byte, error) {
	f.called = true
	_, _ = io.Copy(io.Discard, r) // 消费掉待签名的编码对象
	return f.sig, nil
}

// initRepo 在临时目录初始化一个 git 仓库,返回路径。
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("初始化 git 仓库: %v", err)
	}
	return dir
}

func testIdentity() Identity {
	return Identity{Name: "tally", Email: "tally@example.com", When: time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)}
}

// 正例:签署产出每条条款的内容哈希 + 一个真实签名的 git 提交,文件被冻结进仓库。
func TestSignProducesHashesAndSignedCommit(t *testing.T) {
	repo := initRepo(t)
	doc := cleanDoc()
	raw := Serialize(doc)
	signer := &fakeSigner{sig: []byte("-----BEGIN PGP SIGNATURE-----\nFAKE\n-----END PGP SIGNATURE-----\n")}

	res, err := Sign(SignOptions{
		RepoPath:  repo,
		FileName:  ".tally/requirements/draft.md",
		RawSource: raw,
		Doc:       doc,
		Identity:  testIdentity(),
		Signer:    signer,
	})
	if err != nil {
		t.Fatalf("Sign 出错: %v", err)
	}

	// 每条条款一个内容哈希,且与 ContentHash 一致
	if len(res.ClauseHashes) != len(doc.Clauses) {
		t.Errorf("条款哈希数 = %d, 期望 %d", len(res.ClauseHashes), len(doc.Clauses))
	}
	for _, c := range doc.Clauses {
		if res.ClauseHashes[c.ID] != ContentHash(c) {
			t.Errorf("%s 的哈希与 ContentHash 不一致", c.ID)
		}
	}

	// 签名器真的被调用,提交带上了签名
	if !signer.called {
		t.Error("签名器未被调用")
	}
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("重新打开仓库: %v", err)
	}
	commit, err := r.CommitObject(res.Commit)
	if err != nil {
		t.Fatalf("取提交对象: %v", err)
	}
	if commit.PGPSignature != string(signer.sig) {
		t.Errorf("提交未携带签名: %q", commit.PGPSignature)
	}
	if commit.Author.Name != "tally" || commit.Author.Email != "tally@example.com" {
		t.Errorf("作者身份不符: %+v", commit.Author)
	}
	if !strings.Contains(commit.Message, "clause-hash: REQ-1 "+string(ContentHash(doc.Clauses[0]))) {
		t.Errorf("提交信息未记录内容哈希: %q", commit.Message)
	}

	// 需求文件被写进仓库
	b, err := os.ReadFile(filepath.Join(repo, ".tally", "requirements", "draft.md"))
	if err != nil {
		t.Fatalf("冻结文件未写入: %v", err)
	}
	if string(b) != raw {
		t.Errorf("冻结内容与原文不一致")
	}
}

// 同一文档重复签署,内容哈希确定不变(挂钩才稳)。
func TestSignDeterministicHash(t *testing.T) {
	doc := cleanDoc()
	r1, err := Sign(SignOptions{RepoPath: initRepo(t), FileName: "a.md", Doc: doc, Identity: testIdentity()})
	if err != nil {
		t.Fatalf("Sign1: %v", err)
	}
	r2, err := Sign(SignOptions{RepoPath: initRepo(t), FileName: "a.md", Doc: doc, Identity: testIdentity()})
	if err != nil {
		t.Fatalf("Sign2: %v", err)
	}
	if r1.ClauseHashes["REQ-1"] != r2.ClauseHashes["REQ-1"] {
		t.Error("同一文档两次签署的内容哈希不一致")
	}
}

// 负例:结构缺口是硬闸 —— Lint 还有问题就拒绝签署,且能用 errors.Is 识别。
func TestSignRefusesStructuralGap(t *testing.T) {
	doc := cleanDoc()
	doc.Clauses[0].Criteria = nil // 造一个 L2 缺口

	_, err := Sign(SignOptions{RepoPath: initRepo(t), FileName: "a.md", Doc: doc, Identity: testIdentity()})
	if err == nil {
		t.Fatal("有结构缺口时应拒绝签署")
	}
	if !errors.Is(err, ErrStructuralGap) {
		t.Errorf("应可用 errors.Is 识别 ErrStructuralGap, got %v", err)
	}
	var sge *StructuralGapError
	if !errors.As(err, &sge) {
		t.Fatalf("应能取出结构缺口清单, got %T", err)
	}
	if !findIssue(sge.Issues, "L2", "验收条件") {
		t.Errorf("缺口清单应含 L2 问题, got %v", sge.Issues)
	}
}

// 负例:缺身份不能签。
func TestSignRequiresIdentity(t *testing.T) {
	_, err := Sign(SignOptions{RepoPath: initRepo(t), FileName: "a.md", Doc: cleanDoc()})
	if err == nil {
		t.Fatal("缺身份时应报错")
	}
}

// 无签名器也能签(不签名的提交),结构照建 —— 签名是可选项。
func TestSignWithoutSigner(t *testing.T) {
	repo := initRepo(t)
	res, err := Sign(SignOptions{RepoPath: repo, FileName: "a.md", Doc: cleanDoc(), Identity: testIdentity()})
	if err != nil {
		t.Fatalf("无签名器签署出错: %v", err)
	}
	r, _ := git.PlainOpen(repo)
	commit, err := r.CommitObject(res.Commit)
	if err != nil {
		t.Fatalf("取提交: %v", err)
	}
	if commit.PGPSignature != "" {
		t.Error("未提供签名器时不应有签名")
	}
}

// 无签名签署(经 gitwriter 单一写边界)同样真实冻结:提交对象存在,提交信息
// 记录同一份 clause-hash 清单,树里是同一份文件内容,作者身份来自注入 Identity。
func TestSignViaGitWriterFreezesContentAndHashes(t *testing.T) {
	repo := initRepo(t)
	doc := cleanDoc()
	raw := Serialize(doc)
	res, err := Sign(SignOptions{RepoPath: repo, FileName: ".tally/requirements/draft.md", RawSource: raw, Doc: doc, Identity: testIdentity()})
	if err != nil {
		t.Fatalf("无签名器签署出错: %v", err)
	}
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("重新打开仓库: %v", err)
	}
	commit, err := r.CommitObject(res.Commit)
	if err != nil {
		t.Fatalf("取提交对象: %v", err)
	}
	if !strings.Contains(commit.Message, "clause-hash: REQ-1 "+string(ContentHash(doc.Clauses[0]))) {
		t.Errorf("提交信息未记录内容哈希: %q", commit.Message)
	}
	if commit.Author.Name != "tally" || commit.Author.Email != "tally@example.com" {
		t.Errorf("作者身份不符: %+v", commit.Author)
	}
	f, err := commit.File(".tally/requirements/draft.md")
	if err != nil {
		t.Fatalf("需求文件未冻结进提交树: %v", err)
	}
	if s, err := f.Contents(); err != nil || s != raw {
		t.Errorf("提交树中的冻结内容与原文不一致: %v", err)
	}
}

// 并发基线失配是硬拒绝:调用方按旧 HEAD 编排的提交,在 HEAD 已被并发写入
// 推走后必须被拒(gitwriter.ErrBaseRevision,可用 errors.Is 识别),
// 且原子回滚 —— 工作区不留半写状态(AC2)。
func TestSignConcurrentBaseMismatchRejected(t *testing.T) {
	repo := initRepo(t)
	// 第一笔:未出生 HEAD(基线 "")→ 成功,HEAD = C1
	c1, err := commitFileGitWriter(repo, ".tally/requirements/draft.md", []byte("v1\n"), "seed: v1\n", testIdentity(), "")
	if err != nil {
		t.Fatalf("种子提交出错: %v", err)
	}
	// 第二笔:以当前 HEAD(C1)为基线 → 成功,HEAD 前移到 C2
	if _, err := commitFileGitWriter(repo, ".tally/requirements/draft.md", []byte("v2\n"), "seed: v2\n", testIdentity(), c1.String()); err != nil {
		t.Fatalf("以当前基线提交出错: %v", err)
	}
	// 并发模拟:仍按 C1 编排的提交,此时 HEAD 已是 C2 → 拒绝
	_, err = commitFileGitWriter(repo, ".tally/requirements/draft.md", []byte("v3\n"), "stale: v3\n", testIdentity(), c1.String())
	if err == nil {
		t.Fatal("基线失配时应拒绝提交")
	}
	if !errors.Is(err, gitwriter.ErrBaseRevision) {
		t.Errorf("应可用 errors.Is 识别 gitwriter.ErrBaseRevision, got %v", err)
	}
	// 原子性:被拒的提交既没改文件也没留下提交
	b, rerr := os.ReadFile(filepath.Join(repo, ".tally", "requirements", "draft.md"))
	if rerr != nil {
		t.Fatalf("读回文件: %v", rerr)
	}
	if string(b) != "v2\n" {
		t.Errorf("被拒提交不应改动工作区, got %q", string(b))
	}
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("重新打开仓库: %v", err)
	}
	head, err := r.Head()
	if err != nil {
		t.Fatalf("取 HEAD: %v", err)
	}
	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("取 HEAD 提交: %v", err)
	}
	if !strings.HasPrefix(commit.Message, "seed: v2") {
		t.Errorf("被拒提交不应产生新提交, HEAD = %q", commit.Message)
	}
}

// 同一内容重复签署:gitwriter 无空提交面,不新造提交;「重复签署也留下带
// 哈希的签署记录」由现 HEAD 承载 —— 其提交信息正是同一份 clause-hash 清单,
// 内容哈希确定不变,签署结果仍可追溯。
func TestSignSameContentTwiceRecordsExistingCommit(t *testing.T) {
	repo := initRepo(t)
	doc := cleanDoc()
	raw := Serialize(doc)
	r1, err := Sign(SignOptions{RepoPath: repo, FileName: "a.md", RawSource: raw, Doc: doc, Identity: testIdentity()})
	if err != nil {
		t.Fatalf("Sign1: %v", err)
	}
	r2, err := Sign(SignOptions{RepoPath: repo, FileName: "a.md", RawSource: raw, Doc: doc, Identity: testIdentity()})
	if err != nil {
		t.Fatalf("同内容重复签署不应报错: %v", err)
	}
	if r1.Commit != r2.Commit {
		t.Errorf("同内容重复签署应指向同一签署记录, got %s vs %s", r1.Commit, r2.Commit)
	}
	if r2.ClauseHashes["REQ-1"] != r1.ClauseHashes["REQ-1"] {
		t.Error("同内容重复签署的内容哈希应一致")
	}
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("重新打开仓库: %v", err)
	}
	commit, err := r.CommitObject(r2.Commit)
	if err != nil {
		t.Fatalf("取签署记录: %v", err)
	}
	if !strings.Contains(commit.Message, "clause-hash: REQ-1 "+string(ContentHash(doc.Clauses[0]))) {
		t.Errorf("签署记录未记录内容哈希: %q", commit.Message)
	}
}
