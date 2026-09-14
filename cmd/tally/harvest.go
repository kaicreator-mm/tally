package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"tally/cmd/internal/airuntimelegacy"
	"tally/internal/environment"
	"tally/internal/harvest"
	"tally/internal/runner"
)

// runHarvest 手动触发服务定义抽取:tally harvest [-service a,b] [-repo PATH] [-list]。
//
// 流程:取来源材料 → 通用推理提案(必须含健康检查)→ 实际拉起跑健康检查(经 environment)→
// 通过的写进 catalog/ 并 git commit。三轮不过跳过该服务并记因。
// 清单显式有限(默认十来个),不做全量批处理(PRD 规则 ①)。
//
// 推理走 airuntime 边界(P0-10):域侧只声明 operation/intent/input/budget,
// 传输在这里接线 —— 真实 SDK 接线落地前用 airuntimelegacy adapter 过渡(见 req.go)。
func runHarvest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("harvest", flag.ContinueOnError)
	service := fs.String("service", "", "只抽指定服务(逗号分隔;缺省抽默认清单 ~10 个)")
	repoPath := fs.String("repo", ".", "git 仓库根(catalog/ 在其下)")
	listOnly := fs.Bool("list", false, "只列出将抽取的清单,不执行")
	if err := fs.Parse(args); err != nil {
		return err
	}

	services := harvest.DefaultServices
	if *service != "" {
		services = filterServices(services, strings.Split(*service, ","))
		if len(services) == 0 {
			return fmt.Errorf("没有匹配的服务: %s", *service)
		}
	}

	if *listOnly {
		fmt.Printf("抽取清单(%d 个):\n", len(services))
		for _, s := range services {
			fmt.Printf("  %-12s %s:%s\t%s\n", s.Name, s.Source, s.Ref, s.Hints)
		}
		return nil
	}

	// 前置硬节点:LICENSE-NOTES.md 必须存在(许可证查完才许抽)。
	if _, err := os.Stat(filepath.Join(*repoPath, "catalog", "LICENSE-NOTES.md")); err != nil {
		return fmt.Errorf("缺 catalog/LICENSE-NOTES.md:许可证未查完不允许抽取(T02 硬节点): %w", err)
	}

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	mgr := &environment.Manager{Docker: runner.NewDockerRunner(), Store: st.Queries()}
	fetcher := harvest.HTTPFetcher{}
	h := &harvest.Harvester{
		Executor: airuntimelegacy.FromEnv(),
		Verifier: &envVerifier{mgr: mgr},
		Catalog:  &harvest.GitCatalog{RepoPath: *repoPath, Identity: harvestIdentity()},
		Sources: map[string]harvest.Source{
			"coolify":  harvest.CoolifySource{Fetcher: fetcher},
			"upstream": harvest.HTTPSource{SourceName: "upstream", Fetcher: fetcher},
			"docs":     harvest.HTTPSource{SourceName: "docs", Fetcher: fetcher},
		},
	}

	outcomes := h.Run(ctx, services)
	stored, skipped := 0, 0
	for _, o := range outcomes {
		switch {
		case o.Stored:
			stored++
			fmt.Printf("✓ %s 已验证并写入 catalog/%s\n", o.Service, o.Service)
		case o.Skipped:
			skipped++
			fmt.Printf("✗ %s 跳过: %s\n", o.Service, o.Reason)
		}
	}
	fmt.Printf("\n完成:%d 收录,%d 跳过\n", stored, skipped)
	return nil
}

// filterServices 按名字(逗号分隔)从清单里挑服务;未知名字忽略。
func filterServices(all []harvest.Service, names []string) []harvest.Service {
	want := map[string]bool{}
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			want[n] = true
		}
	}
	var out []harvest.Service
	for _, s := range all {
		if want[s.Name] {
			out = append(out, s)
		}
	}
	return out
}

// harvestIdentity 从环境变量取提交身份,缺省给合理默认。
func harvestIdentity() harvest.Identity {
	name := os.Getenv("TALLY_HARVEST_NAME")
	if name == "" {
		name = "tally-harvest"
	}
	email := os.Getenv("TALLY_HARVEST_EMAIL")
	if email == "" {
		email = "tally@localhost"
	}
	return harvest.Identity{Name: name, Email: email}
}

// envVerifier 把 internal/environment 的拉起能力适配成 harvest.Verifier。
// 这是业务包协作的正确位置:harvest 不 import environment,由 CLI(上层)在这里接线。
type envVerifier struct {
	mgr *environment.Manager
}

// Verify 把 harvest 定义写进临时目录,转成 environment 定义实际拉起,
// 跑健康检查;通过则返回验证证据(时间 + 完整版本组合),最后推倒清理。
func (v *envVerifier) Verify(ctx context.Context, def harvest.Definition) (harvest.VerifyResult, error) {
	dir, err := os.MkdirTemp("", "harvest-verify-*")
	if err != nil {
		return harvest.VerifyResult{}, fmt.Errorf("临时目录: %w", err)
	}
	defer os.RemoveAll(dir)

	composePath := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(def.Compose), 0o600); err != nil {
		return harvest.VerifyResult{}, fmt.Errorf("写临时 compose: %w", err)
	}

	// 用 harvest 定义生成 environment 兼容 YAML(compose_file 指向临时文件),
	// 走 environment.ParseDefinitionBytes:它负责解析时长、算内容哈希(INV-4)。
	ydef := def
	ydef.ComposeFile = composePath
	if ydef.Project == "" {
		ydef.Project = "harvest"
	}
	ydef.ProjectName = "harvest-" + sanitizeProject(def.Name)
	raw, err := yaml.Marshal(ydef)
	if err != nil {
		return harvest.VerifyResult{}, fmt.Errorf("序列化环境定义: %w", err)
	}
	envDef, err := environment.ParseDefinitionBytes(raw)
	if err != nil {
		return harvest.VerifyResult{}, fmt.Errorf("解析环境定义: %w", err)
	}

	res, err := v.mgr.Up(ctx, envDef)
	if err != nil {
		return harvest.VerifyResult{}, fmt.Errorf("拉起验证: %w", err)
	}
	// 清理用独立于 ctx 的背景上下文:即便验证 ctx 已取消也要推倒,别留容器。
	defer func() { _ = v.mgr.Down(context.Background(), envDef) }() //nolint:contextcheck // 故意:清理须独立于可能已取消的 ctx

	return harvest.VerifyResult{
		VerifiedAt: time.Now(),
		Combo:      res.Combo,
		Record:     healthRecord(res.Health),
	}, nil
}

// healthRecord 把健康检查汇总压成机器记录(如 "health=3/3ok")。
func healthRecord(rep environment.HealthReport) string {
	ok := 0
	for _, r := range rep.Results {
		if r.OK {
			ok++
		}
	}
	return fmt.Sprintf("health=%d/%dok", ok, len(rep.Results))
}

// sanitizeProject 把服务名压成合法 compose 项目名(小写字母数字与连字符)。
func sanitizeProject(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
