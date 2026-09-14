# P6-06 Execution Report — Safe project export

- **Status:** DONE
- **Date:** 2026-09-12
- **Validation HEAD:** master `f7211d35ff95e0a93c0f49de726896ab27e18016`(会话期间 P6-05/P6-01 相继落地,最终验证均在 `f7211d3` 上执行);任务基线 `5c08b3d`;审计源基线 `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`
- **Kind:** implementation(scripts/** 内新增独立工具,零 Go 生产代码改动)

## 0. 语义结论

`SOURCE_BASELINE.md` 已知生产缺口 #7 —— "repository export currently includes local/runtime artifacts when the entire worktree is archived"(现状即对整个工作树整体打包,`.codegraph/`、`.codex/`、`.e2e/`、`.worktrees/`、`bin/` 全部是盘上存在但被 .gitignore 忽略的目录)—— 由本任务闭合:

新增 `scripts/export`(Tally Go 内 `go run ./scripts/export`,与 `scripts/preflight` 同构的 scripts 内 Go 工具),产出**确定性、钉定单一 source commit 的源码/文档导出包**(供外部代码评审 / agent 分析),不含任何本地运行态与凭证:

```text
git -C <repo> rev-parse --verify <ref>^{commit}   // 钉定 source commit(默认 HEAD)
git -C <repo> archive --format=tar <commit>       // 只取已提交树,绝不读工作树
  → manifest/denylist 全量扫描(fail-safe)
  → 扫描干净才发布 archive + manifest;任一命中则零发布 + 类型化错误
```

安全性质(任务契约 §Implementation requirements 1 + L3 Failure Handling "deny secrets/runtime dirs even if `.gitignore` changes"):

1. **内容只来自 Git 对象库**——被忽略的本地运行态目录按构造不可能进入导出;
2. **第二道 denylist 扫描对 archive 本身执行**,即使违禁内容被 git 跟踪(例如 .gitignore 被改动/删除后提交)也拒绝发布:deny wins over ignore;
3. denylist 三类规则(全部逐项对照当前 HEAD 跟踪树验证过**零误伤**,列表与验证日期写在代码注释里):
   - 目录段(运行态/构建/缓存/本地凭证):`.git`、`.worktrees`、`.e2e`、`.codegraph`、`.codex`、`bin`、`node_modules`、`__pycache__`、`.pytest_cache`、`.mypy_cache`、`.ruff_cache`、`.cache`、`.venv`、`venv`、`.gradle`、`.turbo`、`.next`、`.nuxt`、`.nyc_output`、`.ssh`、`.aws`、`.gnupg`;
   - 文件名(凭证/机密/OS 垃圾):`.env`、`.env.*`(唯一豁免 `.env.example`,镜像 .gitignore 的 `!.env.example`)、`id_rsa/id_dsa/id_ecdsa/id_ed25519*`、`*.pem`、`*.key`、`*.pfx`、`*.p12`、`*.jks`、`*.kdbx`、`*.keystore`、`.netrc`、`.npmrc`、`.pypirc`、`.htpasswd`、`.DS_Store`、`Thumbs.db`;
   - 内容级兜底:正则文件流式扫描 PEM 私钥标记(`-----BEGIN … PRIVATE KEY-----` 七种,含跨读取块边界的 carry 处理)。**有意不**做 AWS AKIA 内容扫描:`internal/logpipe/redact_test.go` 以 AWS 官方文档公开示例 key 作为必备源码夹具,扫了必然误伤必备源(已在包注释与本报告记录该边界);
4. **路径安全**:拒绝 symlink/hardlink/设备等一切非"正则文件+目录"条目、绝对路径、`\` 分隔符、`..` 段、重复条目名(导出是源码投影,不是文件系统镜像);
5. **确定性 + source-commit 身份保留(DL-04 读法)**:同一 commit 恒定产出字节级相同的 archive 与 manifest(测试钉死);manifest 记录完整 commit sha、请求 ref、archive sha256/size、逐文件 path/size/sha256、`denied_count:0`;git archive 自身的 `pax_global_header`(comment=commit sha)保留在包内作为包内 provenance,测试断言之;
6. **fail-safe 发布纪律**:先写临时文件、全量扫描通过才原子改名发布;命中 deny 时临时文件清除、输出目录零残留、返回 `*DeniedPathError`(CLI exit 2,逐条列出 path/rule/reason);空导出拒绝发布;无法解析 ref 是操作性错误(exit 1)而非 deny、更不是静默成功。

## 1. Changed files(全部在 Allowed Paths:scripts/**、Makefile、docs/**)

- `scripts/export/export.go`(新)— 核心:`RunExport`(钉 commit → 流式 `git archive` → 扫描 → 原子发布)+ denylist 表 + 纯 tar 巡查器 `inspectEntries`/`evaluateEntry` + 内容级 secretScanner + manifest 写发。结构化 argv(无 shell),`GIT_TERMINAL_PROMPT=0`。
- `scripts/export/main.go`(新)— CLI:`-repo/-ref/-out`(默认 `<repo>/bin/export`,自身就在 deny 目录里,不产生自包含污染);exit 0/1/2/3 语义文件头文档化。
- `scripts/export/export_test.go`(新)— 8 个聚焦测试(含负路径,见 §3)。
- `Makefile`(改)— 新 `export-src` 目标(入 help 与 .PHONY),一行 `go run ./scripts/export`。
- `docs/tasks/v1.4.1/reports/P6-06_safe_project_export.md`(本报告)。

`scripts/v14_trial/**` 零触碰(其 LoopConfig 组合归 P4-03/P4-07 所有);`test/**` 零触碰(并行 lane P6-04 所有权);`.gitignore` 零触碰(不在 Allowed Paths)。

## 2. Semantic changes

- Tally 既有生产语义**零改动**:无 internal/**、cmd/** 改动;新增工具是独立 scripts 包,不进任何生产组合根(导出本身按定义就是只读投影)。
- 新行为 = 本工具 + `make export-src` 操作者入口。

## 3. Tests / commands executed(exact)

| 命令 | 结果 |
|---|---|
| `go test ./scripts/export/... -count=1 -v` | **PASS 8/8**(~3.1s):real-repo 验收测试 / 确定性 / commit 钉定 / 提交违禁 fail-safe / 内容级机密 fail-safe / 未知 ref / 合成 tar 巡查器(含 32KiB 边界跨块标记)/ `.env.example` 豁免 |
| `go test ./scripts/... -count=1` | PASS(preflight 邻居不回退) |
| `gofmt -l scripts/export/` | clean |
| `go vet ./scripts/...` | PASS |
| `go run ./scripts/export`(真实 CLI,验证 HEAD `f7211d3`) | PASS,exit 0:`1313 files, archive_sha256=461f6b3b7750…`,manifest 落盘,grep 证 0 条 denied 前缀路径(验证产物已清理) |
| `go build -o …/export-cli.exe ./scripts/export` + 对投毒临时库(已提交 `bin/tool`、`.e2e/s.json`)直接执行 | **exit=2**,逐条列出 denials,输出目录零文件;同二进制对真实库 exit=0(exit 契约实测钉死;注:`go run` 包装层在本工具链把子进程非零退出转发为自身 1 并打印 "exit status 2",直接消费方请用构建出的二进制) |
| `git diff --check` | clean(两条 CRLF 警告来自并行 lane P2-01 的 `internal/web`/`web/templates` 在途文件,非本任务) |
| 新文件 CRLF 检查(`grep \r`) | 全部 LF_ONLY(遵守 .gitattributes eol=lf) |
| `make export-src` / `make check` | NOT_RUN(reason:本机 Git Bash 无 make;目标文本与既有 `go run ./cmd/tally migrate` 同构) |
| repo-wide sweeps / integration 套件 | NOT_RUN(reason:orchestrator 规则禁宽扫描;本任务无 DB 面) |

## 4. Baseline failures observed

None。`scripts/preflight` 套件在改动在场时保持 PASS。

## 5. 验收对照(任务契约)

- **AC — Generated archive contains required source/docs and zero denied paths:PASS。** `TestExportRealRepoContainsRequiredSourcesAndZeroDeniedPaths` 对真实库 HEAD 双向验证:重开已发布 tar 逐条巡查(类型、集合与 manifest 精确一致)+ 独立于扫描器实现的任务点名 denylist 复查(0 命中)+ 必备面断言(`go.mod`、`go.sum`、`Makefile`、`.gitattributes`、本任务文件、`scripts/preflight/main.go`,以及 `cmd/ internal/ migrations/ docs/ scripts/ web/ test/ runtimes/` 每树 ≥1 条)。
- **Verification — Automated manifest/denylist scan:PASS。** 即上述测试;另附真实 CLI 与投毒库 CLI 实测。
- **Implementation requirement 2 — Archive inspection test:PASS**(§3 第 1 行)。

## 6. Remaining risks

- denylist 为**名称/标记级**(文件名 + PEM 私钥内容标记),不是内容 DLP 引擎:未扫描的机密形态(如嵌入式 API key 字符串、AWS key id——因公开示例夹具而有意排除)不设防;未来若跟踪树新增与 deny 目录段同名的合法路径,导出会按设计 fail-safe 失败,需有意识修订列表。
- 确定性承诺范围 = 同一 commit + 同一 git 版本(git archive 的 tar 头取 commit 时间戳、`eol=lf` 属性归一);跨 git 大版本的 pax 头形态差异未钉。
- 导出默认取 HEAD:**未提交的在途编辑不入包**(设计使然,与 DL-04 单一钉定源 commit 一致);需要导出未提交状态时请先提交。
- `export-src` 目标未经 `make` 实跑(环境无 make),仅有逐字同构性佐证。

## 7. Decision Locks

未触碰、未绕过。DL-04(单一钉定源 commit)按"导出即从钉定 commit 解析"读法满足并由测试钉死;DL-17(held-out)全程未搜索/未挂载/未读;共享热点零触碰;DECISION_LOCKS/PRD/STATUS/TASK_DAG/PATH_OWNERSHIP/MODEL_ASSIGNMENT 零改动。

## 8. Completion report(task contract format)

1. **status:** DONE
2. **changed files:** §1(三个新文件 + Makefile 一处增补 + 本报告)
3. **semantic changes:** 生产语义零改动;新增只读导出工具与 `export-src` 目标(§2)
4. **tests/commands:** §3(全部 PASS/NOT_RUN 如实记录,exit 契约实测)
5. **baseline failures:** none(§4)
6. **remaining risks:** §6
7. **Decision Locks touched:** none(§7)
8. **commit:** not committed(orchestration 规则:本 agent 非 committer)

## 9. L3 completion evidence

- **exact source/tests read:** 任务文件全文(含 L3 Pack);`docs/tasks/v1.4.1/DECISION_LOCKS.md` 全文、`docs/tasks/v1.4.1/PRD_AMENDMENTS.md` 全文、`docs/tasks/v1.4.1/SOURCE_BASELINE.md` 全文(缺口 #7 即本任务靶点);当前源:`Makefile`、`.gitignore`、`.gitattributes`、`scripts/preflight/{main.go,preflight.go}`(scripts 内 Go 工具 + go test 同构先例)、`git ls-files` 对跟踪树的 denylist 误伤普查(零命中)、非正则条目普查(零 symlink/submodule)、`go.mod` 依赖白名单面;`docs/tasks/v1.4.1/reports/P6-04_…md`(报告结构对齐)。
- **external reference used:** 仅 L3 Pack 内的 EXT-GIT-WORKTREE(git-worktree 手册,语义参考,零代码复制);另就地实测了本机 git 2.55 的 `git archive` pax_global_header 形态(comment=commit sha,Go tar.Reader 将其解析进 `PAXRecords["comment"]` 并消费数据段)。
- **narrowest seam 论证:** 最小有效缝 = `git archive` 钉定 commit(权威生产者:内容天然来自对象库,被忽略运行态按构造不可入包)+ 一层对产物的 manifest/denylist 扫描(补齐"即使被跟踪也拒绝"的 fail-safe 面)+ 原子发布纪律。未建并行导出子系统、未加依赖(纯标准库 archive/tar+crypto/sha256+os/exec)、未动生产组合根;`scripts/preflight` 已确立 scripts/** 内 Go 工具 + go test 的先例,工具落位与其同构。
- **negative tests added:** 提交违禁路径(`bin/tool.exe`、`sub/.env`、`.e2e/state.json`)→ `*DeniedPathError` + 零发布;内容级私钥标记(无辜文件名)→ `secret-content` deny;合成 tar 的 symlink/`..`/绝对路径/反斜杠/重复条目/跨块 PEM 标记逐类拒绝;未知 ref → 操作性错误且非 deny;`.env.production` deny 与 `.env.example` 豁免双向钉死。
- **known uncertainty / version sensitivity:** §6(git 版本间 pax 头形态、`go run` 包装层退出码转发、make 缺失)。
- **Decision Locks not bypassed:** 确认(§7)。
