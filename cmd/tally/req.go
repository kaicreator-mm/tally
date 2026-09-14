package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"

	"tally/cmd/internal/airuntimelegacy"
	"tally/internal/requirement"
)

// runReq 是 `tally req` 子命令入口:lint / interview / sign。
func runReq(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: tally req <lint|interview|sign> <file>")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "lint":
		return runReqLint(rest)
	case "interview":
		return runReqInterview(ctx, rest)
	case "sign":
		return runReqSign(rest)
	default:
		return fmt.Errorf("未知子命令 req %s(应为 lint/interview/sign)", sub)
	}
}

// runReqLint 跑五闸,打印结构化问题清单;有缺口时以非零退出码收尾(供 CI 把关)。
func runReqLint(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: tally req lint <file>")
	}
	doc, err := requirement.ParseFile(args[0])
	if err != nil {
		return err
	}
	issues := requirement.Lint(doc)
	if len(issues) == 0 {
		fmt.Println("干净:无结构缺口")
		return nil
	}
	for _, i := range issues {
		fmt.Println(i.String())
	}
	return fmt.Errorf("有 %d 个结构缺口未清", len(issues))
}

// runReqInterview 让通用推理把实质缺口转成问题。结构缺口先报出来(那是签署硬闸)。
//
// 推理走 airuntime 边界(P0-10):域侧只声明 operation/intent/input/budget,
// 传输在这里接线 —— 真实 SDK 接线落地前用 airuntimelegacy adapter 过渡(见其注释)。
func runReqInterview(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("用法: tally req interview <file>")
	}
	doc, err := requirement.ParseFile(args[0])
	if err != nil {
		return err
	}
	if issues := requirement.Lint(doc); requirement.HasStructuralGap(issues) {
		fmt.Println("注意:仍有结构缺口(签署硬闸),先清掉:")
		for _, i := range issues {
			fmt.Println("  " + i.String())
		}
	}
	aiExec := airuntimelegacy.FromEnv()
	qs, err := requirement.Interview(ctx, aiExec, doc)
	if err != nil {
		return err
	}
	if len(qs) == 0 {
		fmt.Println("问询未发现实质缺口。")
		return nil
	}
	fmt.Println("实质缺口建议(可驳回,不阻塞签署):")
	for n, q := range qs {
		fmt.Printf("%d. [%s] %s\n", n+1, q.Gap, q.Question)
		if len(q.Candidates) > 0 {
			fmt.Printf("   候选: %s\n", strings.Join(q.Candidates, " / "))
		}
	}
	return nil
}

// runReqSign 冻结需求:产出内容哈希并写 git 提交。配了 TALLY_SIGN_KEY 时用外部 gpg 签名。
func runReqSign(args []string) error {
	fs := flag.NewFlagSet("req sign", flag.ContinueOnError)
	name := fs.String("name", os.Getenv("TALLY_SIGN_NAME"), "签署人姓名(或 env TALLY_SIGN_NAME)")
	email := fs.String("email", os.Getenv("TALLY_SIGN_EMAIL"), "签署人邮箱(或 env TALLY_SIGN_EMAIL)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("用法: tally req sign <file> [--name N --email E]")
	}
	file := fs.Arg(0)

	doc, err := requirement.ParseFile(file)
	if err != nil {
		return err
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("解析路径: %w", err)
	}
	repoRoot, err := findRepoRoot(filepath.Dir(absFile))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(repoRoot, absFile)
	if err != nil {
		return fmt.Errorf("计算仓库内相对路径: %w", err)
	}
	fileName := filepath.ToSlash(rel)

	id, err := resolveIdentity(repoRoot, *name, *email)
	if err != nil {
		return err
	}

	var signer git.Signer
	if key := os.Getenv("TALLY_SIGN_KEY"); key != "" {
		signer = gpgSigner{keyID: key}
	}

	raw, err := os.ReadFile(absFile)
	if err != nil {
		return fmt.Errorf("读需求文件: %w", err)
	}
	res, err := requirement.Sign(requirement.SignOptions{
		RepoPath:  repoRoot,
		FileName:  fileName,
		RawSource: string(raw),
		Doc:       doc,
		Identity:  id,
		Signer:    signer,
	})
	if err != nil {
		return err
	}
	signed := "未签名"
	if signer != nil {
		signed = "已签名(gpg:" + os.Getenv("TALLY_SIGN_KEY") + ")"
	}
	fmt.Printf("提交: %s(%s)\n", res.Commit, signed)
	for _, c := range doc.Clauses {
		fmt.Printf("  %s %s\n", c.ID, res.ClauseHashes[c.ID])
	}
	return nil
}

// findRepoRoot 从 start 向上找含 .git 的目录。
func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未在 %s 之上找到 git 仓库(.git)", start)
		}
		dir = parent
	}
}

// resolveIdentity 取签署身份:显式参数 > git 配置。
func resolveIdentity(repoRoot, name, email string) (requirement.Identity, error) {
	if name == "" || email == "" {
		repo, err := git.PlainOpen(repoRoot)
		if err == nil {
			if cfg, err := repo.Config(); err == nil {
				if name == "" {
					name = cfg.User.Name
				}
				if email == "" {
					email = cfg.User.Email
				}
			}
		}
	}
	if name == "" || email == "" {
		return requirement.Identity{}, fmt.Errorf("缺签署身份:用 --name/--email 或配置 git user.name/user.email")
	}
	return requirement.Identity{Name: name, Email: email}, nil
}

// gpgSigner 用外部 gpg 程序签名(go-git 的 git.Signer)。
//
// 走 gpg 二进制而不是 Go 的 openpgp 库:不引入白名单外依赖(INV-6),
// 且与 git 自身签名方式一致。key 由 keyID 指定(TALLY_SIGN_KEY)。
type gpgSigner struct{ keyID string }

func (g gpgSigner) Sign(r io.Reader) ([]byte, error) {
	payload, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读待签名内容: %w", err)
	}
	cmd := exec.Command("gpg", "--status-fd=1", "-bsau", g.keyID)
	cmd.Stdin = bytes.NewReader(payload)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg 签名失败: %w: %s", err, out.String())
	}
	return out.Bytes(), nil
}

// Legacy LLM transport is centralized in cmd/internal/airuntimelegacy.
