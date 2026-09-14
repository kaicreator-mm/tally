package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"tally/internal/environment"
	"tally/internal/ingest"
	"tally/internal/runner"
	"tally/internal/store"
)

// runEnv 手动触发环境操作:tally env <up|down|rebuild> -def <环境定义.yaml> [-project P] [-name N]。
// 拉起/重建会写一行 environment_run(事实面,只追加);推倒不写(environment_run 只记拉起)。
func runEnv(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("用法: tally env <up|down|rebuild> -def <环境定义.yaml> [-project P] [-name N]")
	}
	action, rest := args[0], args[1:]

	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	project := fs.String("project", "", "项目 id(覆盖定义里的 project)")
	name := fs.String("name", "", "环境名(覆盖定义里的 name)")
	defPath := fs.String("def", "", "环境定义文件路径(.tally/environments/*.yaml)")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *defPath == "" {
		return fmt.Errorf("需要 -def 指定环境定义文件")
	}

	def, err := environment.ParseDefinition(*defPath)
	if err != nil {
		return err
	}
	if *project != "" {
		def.Project = *project
	}
	if *name != "" {
		def.Name = *name
	}

	// 注册硬校验:缺健康检查(ErrNoHealthCheck)等直接拒绝,不是警告。
	if err := environment.Validate(def); err != nil {
		return fmt.Errorf("环境定义校验不过: %w", err)
	}

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	m := &environment.Manager{
		Docker: runner.NewDockerRunner(),
		Store:  st.Queries(),
	}
	if def.Kind == environment.KindSSHScript {
		m.SSH = runner.NewSSHRunner(environment.SSHRunnerConfig(def.SSH))
	}

	switch action {
	case "up":
		res, err := m.Up(ctx, def)
		if err != nil {
			ingestEnvFailure(ctx, st, err)
			return err
		}
		fmt.Printf("环境 %s/%s 拉起成功:run=%d 定义=%s\n版本组合: %v\n",
			def.Project, def.Name, res.RunID, res.Definition, res.Combo)
	case "down":
		if err := m.Down(ctx, def); err != nil {
			return err
		}
		fmt.Printf("环境 %s/%s 已推倒(卷/网络已清)\n", def.Project, def.Name)
	case "rebuild":
		res, err := m.Rebuild(ctx, def)
		if err != nil {
			ingestEnvFailure(ctx, st, err)
			return err
		}
		fmt.Printf("环境 %s/%s 重建成功:run=%d 定义=%s\n", def.Project, def.Name, res.RunID, res.Definition)
	default:
		return fmt.Errorf("未知动作 %q(支持 up / down / rebuild)", action)
	}
	return nil
}

// ingestEnvFailure 把拉起失败(*environment.UpError)经 ingest 录进 failure_event。
// 与 exec 编排层同一套录入(D2):CLI 手动触发的失败也进生产链。尽力而为,出错不遮盖原始失败。
func ingestEnvFailure(ctx context.Context, st *store.Store, err error) {
	var uerr *environment.UpError
	if !errors.As(err, &uerr) {
		return
	}
	if _, _, ingErr := ingest.RecordFailure(ctx, st, ingest.Input{
		ProjectID:        uerr.Project,
		Source:           "environment",
		EnvironmentRunID: uerr.RunID,
		Processed:        &uerr.Processed,
		RawLogBytes:      uerr.RawBytes,
		Combo:            uerr.Combo,
		OccurredAt:       uerr.OccurredAt,
	}); ingErr != nil {
		fmt.Printf("提示:录入失败事件出错(environment_run 已记): %v\n", ingErr)
	}
}
