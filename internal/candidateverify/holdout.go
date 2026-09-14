package candidateverify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/holdout"
	"tally/internal/runner"
	"tally/internal/verification"
)

// holdoutBoundary 构造 Tally 控制的 held-out 执行边界(P1-29 的控制面接线):
// held-out 内容只以内存字节到达这里,被物化进一个 Tally 控制的一次性目录
// (os 临时区,在项目 repo 与所有一次性 worktree 之外),冻结命令在该目录内
// 执行;原始输出只在本边界内消化,对外只返回结构化 status + 输出摘要(§14.3)。
// 约定:内容物化为边界目录下的 holdout-input 文件;冻结命令的相对 cwd 在边界
// 目录内解析,越出即拒绝(与 visible 命令的 workspace 纪律一致)。
func holdoutBoundary(rt runner.Runner) verification.HoldoutRunFunc {
	return func(ctx context.Context, req verification.HoldoutRunRequest) (verification.HoldoutRunOutput, error) {
		if req.Command == nil {
			return verification.HoldoutRunOutput{}, errors.New("candidateverify: held-out 冻结命令缺失")
		}
		dir, err := os.MkdirTemp("", "tally-holdout-")
		if err != nil {
			return verification.HoldoutRunOutput{}, fmt.Errorf("candidateverify: 建 holdout 边界目录: %w", err)
		}
		defer os.RemoveAll(dir)
		if err := os.WriteFile(filepath.Join(dir, "holdout-input"), req.Content, 0o600); err != nil {
			return verification.HoldoutRunOutput{}, fmt.Errorf("candidateverify: 物化 held-out 内容: %w", err)
		}
		cwd := dir
		if d := req.Command.Dir; d != "" {
			rel := filepath.Clean(filepath.FromSlash(d))
			if rel == ".." || filepath.IsAbs(rel) ||
				strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return verification.HoldoutRunOutput{}, fmt.Errorf("candidateverify: held-out 冻结命令 cwd %q 越出边界目录,拒绝执行", d)
			}
			if rel != "." {
				cwd = filepath.Join(dir, rel)
				if err := os.MkdirAll(cwd, 0o700); err != nil {
					return verification.HoldoutRunOutput{}, fmt.Errorf("candidateverify: 建 holdout cwd: %w", err)
				}
			}
		}
		if timeout := time.Duration(req.Command.Timeout); timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		res, err := rt.Run(ctx, runner.Command{
			Argv: append([]string(nil), req.Command.Argv...),
			Dir:  cwd,
			Env:  append([]string(nil), req.Command.Env...),
		})
		if err != nil {
			return verification.HoldoutRunOutput{}, err
		}
		status := verification.EvidenceStatusFail
		if res.ExitCode == 0 {
			status = verification.EvidenceStatusPass
		}
		return verification.HoldoutRunOutput{
			Status:       status,
			OutputDigest: holdout.ContentDigest([]byte(res.Stdout + "\n" + res.Stderr)),
		}, nil
	}
}
