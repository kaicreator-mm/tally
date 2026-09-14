-- +goose Up

-- 失败复现上下文的持久化合同从「workdir/repro 文本 + other_targets 字符串」
-- 迁到结构化 CommandSpec(PRD §21.6 Structured Repro:不用 shell string +
-- strings.Fields 表示 repro)。写入方落 internal/command.CommandSpec 的规范
-- JSON(字段集固定为 argv/cwd/env/timeout/shell,shell 恒显式、缺省即 false):
--
--   {"argv":["go","test","./internal/jobs/..."],"cwd":".","env":{},"timeout":"120s","shell":false}
--
-- INV-FACT-01:failure_event 只追加,本迁移只加列、给默认值,不回写任何
-- 存量行(表上的 UPDATE/DELETE 触发器同样禁止回写)。存量行的新列取默认:
-- repro_command 为 JSON null(「未记录结构化复现」哨兵,不是合法 spec)。
--
-- 0013 的旧三列 workdir/repro/other_targets 原样保留,不删不改:它们是存量行
-- 兼容读取的唯一来源。兼容路径定义在 failure_event.sql 的读取合同里:
-- repro_command 为 null 时按旧执行语义重建
-- CommandSpec{Argv: strings.Fields(repro), Cwd: workdir, Shell: false}
-- (0013 时代的 worker 正是 strings.Fields(repro) + Dir(workdir) 执行的);
-- other_target_commands 为空数组时回退 other_targets 里的目标名(无命令)。
-- 新写入只走结构化两列,不再落文本。

-- 结构化复现命令:CommandSpec 规范 JSON;JSON null = 未记录(存量行默认值)。
ALTER TABLE failure_event ADD COLUMN repro_command JSONB NOT NULL DEFAULT 'null';

-- 形状护栏:只能是 null 哨兵或 JSON 对象,禁止退化成 shell 字符串(旧合同)。
ALTER TABLE failure_event ADD CONSTRAINT failure_event_repro_command_shape
  CHECK (jsonb_typeof(repro_command) IN ('null', 'object'));

-- 结构化其余构建目标(G3 不受影响检查):JSON 数组,元素为
-- {"target": "<目标名>", "command": <CommandSpec>}。空数组 = 未记录结构化目标。
ALTER TABLE failure_event ADD COLUMN other_target_commands JSONB NOT NULL DEFAULT '[]';

-- 形状护栏:必须是 JSON 数组。
ALTER TABLE failure_event ADD CONSTRAINT failure_event_other_target_commands_shape
  CHECK (jsonb_typeof(other_target_commands) = 'array');

-- +goose Down
ALTER TABLE failure_event DROP CONSTRAINT IF EXISTS failure_event_other_target_commands_shape;
ALTER TABLE failure_event DROP COLUMN IF EXISTS other_target_commands;
ALTER TABLE failure_event DROP CONSTRAINT IF EXISTS failure_event_repro_command_shape;
ALTER TABLE failure_event DROP COLUMN IF EXISTS repro_command;
