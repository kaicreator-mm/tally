package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"tally/internal/core"
	"tally/internal/deploy"
	"tally/internal/ingest"
	"tally/internal/runner"
	"tally/internal/store"
)

// runDeploy 手动触发部署 / 回滚:
//
//	tally deploy [-def <部署定义.yaml>] [-target-env N]            部署定义里的制品
//	tally deploy rollback -def <部署定义.yaml> [-target-env N] [-digest D]
//
// 每次写一行 deploy_run(事实面,只追加)。-target-env 覆盖定义里的目标 ——
// 重跑验证时用它把同一个 digest 部到隔离环境,不复用正在使用的测试环境。
func runDeploy(ctx context.Context, args []string) error {
	rollback := false
	if len(args) > 0 && args[0] == "rollback" {
		rollback, args = true, args[1:]
	}

	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	defPath := fs.String("def", "", "部署定义文件路径(.tally/deploys/*.yaml)")
	targetEnv := fs.String("target-env", "", "目标环境名(覆盖定义里的 target_env;验证用隔离环境)")
	digest := fs.String("digest", "", "制品 digest(覆盖定义里的 artifact_digest;回滚到指定版本用)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *defPath == "" {
		return fmt.Errorf("需要 -def 指定部署定义文件")
	}

	def, err := deploy.ParseDefinition(*defPath)
	if err != nil {
		return err
	}
	if *targetEnv != "" {
		def.TargetEnv = *targetEnv
	}
	if *digest != "" {
		def.Digest = core.ArtifactDigest(*digest)
	}
	if err := def.Validate(); err != nil {
		return fmt.Errorf("部署定义校验不过: %w", err)
	}

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	d := &deploy.Deployer{Exec: runner.LocalRunner{}, Store: st.Queries()}
	if rollback {
		res, err := d.Rollback(ctx, def)
		if err != nil {
			ingestDeployFailure(ctx, st, err)
			return err
		}
		fmt.Printf("回滚完成(%s):run=%d 目标 digest=%s 环境=%s/%s\n",
			res.Status, res.RunID, def.Digest, def.Project, def.TargetEnv)
		return nil
	}
	res, err := d.Deploy(ctx, def)
	if err != nil {
		ingestDeployFailure(ctx, st, err)
		return err
	}
	fmt.Printf("部署完成(%s):run=%d digest=%s 环境=%s/%s\n版本组合: %v\n",
		res.Status, res.RunID, def.Digest, def.Project, def.TargetEnv, res.Combo)
	return nil
}

// ingestDeployFailure 把部署/回滚失败(*deploy.FailureError)经 ingest 录进 failure_event。
// 与 exec 编排层同一套录入(D2):CLI 手动触发的失败也进生产链。尽力而为,出错不遮盖原始失败。
func ingestDeployFailure(ctx context.Context, st *store.Store, err error) {
	var ferr *deploy.FailureError
	if !errors.As(err, &ferr) {
		return
	}
	if _, _, ingErr := ingest.RecordFailure(ctx, st, ingest.Input{
		ProjectID:   ferr.Project,
		Source:      "deploy",
		DeployRunID: ferr.RunID,
		Processed:   &ferr.Processed,
		RawLogBytes: ferr.RawBytes,
		Combo:       ferr.Combo,
		OccurredAt:  ferr.OccurredAt,
	}); ingErr != nil {
		fmt.Printf("提示:录入失败事件出错(deploy_run 已记): %v\n", ingErr)
	}
}
