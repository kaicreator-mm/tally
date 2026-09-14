// Command tally 是 tally 的运维 CLI:数据库迁移、手动触发、重算索引、健康检查。
//
// 无状态;状态都在 Postgres 和 git 里。
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "migrate":
		err = runMigrate(ctx)
	case "reindex":
		err = runReindex(ctx, os.Args[2:])
	case "reindex-fingerprints":
		err = runReindexFingerprints(ctx, os.Args[2:])
	case "scan":
		err = runScan(ctx, os.Args[2:])
	case "ingest-build":
		err = runIngestBuild(ctx, os.Args[2:])
	case "deps":
		err = runDeps(ctx, os.Args[2:])
	case "env":
		err = runEnv(ctx, os.Args[2:])
	case "deploy":
		err = runDeploy(ctx, os.Args[2:])
	case "harvest":
		err = runHarvest(ctx, os.Args[2:])
	case "req":
		err = runReq(ctx, os.Args[2:])
	case "report":
		err = runReport(ctx, os.Args[2:])
	case "health":
		err = runHealth(ctx)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tally:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `tally —— 运维 CLI

用法:
  tally migrate                  跑数据库迁移(幂等)
  tally reindex [-repo P -project ID] [-check]
                                       drop 索引面并从 git + 事实面重算;-check 只 diff 不写(CI 自检)
  tally reindex-fingerprints     用留存的原始日志重算历史指纹(RuleVersion 参与)
  tally scan [-toolchain X] <目录>  批量扫日志,输出碰撞率实验结果
  tally ingest-build -project P -commit SHA [-toolchain X] [-log <文件>] [-status failed|passed]
                                       [-workdir D] [-repro CMD] [-targets C1,C2]
                                       录入一次构建:失败时同时写 build_run + failure_event
                                       (缺省读 stdin 当日志,接上 PRD 生产链:失败 → 指纹 → 诊断)
                                       -workdir/-repro/-targets 是复现上下文,诊断 G1/G2/G3 用
  tally deps -project P -repo PATH [-from REV] [-to REV] [-content-repo PATH] [-no-retire]
                                       采集依赖升级事实:逐提交 diff 锁文件写 dependency_change
                                       (重跑安全;失败时间线与依赖升级对齐用)
                                       采完顺带跑修复退休检查:变更碰过的已验证修复重跑复现命令,
                                       不再复现的退休(RETIRED.md + 清索引面);-no-retire 可关
  tally env <up|down|rebuild> -def <环境定义.yaml> [-project P] [-name N]
                                       手动拉起/推倒/重建环境(写 environment_run)
  tally deploy -def <部署定义.yaml> [-target-env N]
                                       部署已构建的制品(写 deploy_run;调 kamal/ansible/compose)
  tally deploy rollback -def <部署定义.yaml> [-target-env N] [-digest D]
                                       回滚到指定 digest(调工具自带回滚,写 deploy_run)
  tally harvest [-service a,b] [-repo PATH] [-list]
                                       抽取服务定义:LLM 提案(须含健康检查)→ 实际拉起验证 → 写 catalog/
                                       -list 只列清单;缺省抽默认清单(~10 个),不做全量批处理
  tally req lint <file>          需求契约五闸检查(结构化问题清单)
  tally req interview <file>     LLM 把实质缺口转成问题(带候选答案)
  tally req sign <file>          签署:产出内容哈希 + git 提交(TALLY_SIGN_KEY 时 gpg 签名)
  tally health                   检查数据库连通性
  tally report baseline [-task ID]
                                       度量基线报告(P0-08):从 human_effort/llm_usage 真实历史
                                       组装只读基线读数;缺失维度显式「样本不足」,不补 0

环境变量:
  DATABASE_URL     Postgres 连接串(默认 postgres://postgres:postgres@localhost:5432/tally?sslmode=disable)
`)
}
