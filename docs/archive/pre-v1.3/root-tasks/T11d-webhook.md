# T11d 运输层:webhook 接收构建日志

## 前置

T11b(webhook 要能携带复现上下文三字段)。

## 目标

把「日志怎么进 tally」从纯手动 CLI 往前推一步:server 暴露一个
**token 鉴权的 webhook**,CI 构建失败时 POST 一段 JSON(构建元数据 + 日志全文)
即完成 `IngestBuild`。PRD 第三部分「从哪收」的 CI 侧接入点;
「CI 平台 API 主动拉取」仍不做(那是各平台适配,成本高收益低,记 BACKLOG)。

## 已锁定的设计决策

- **D-d1 鉴权 = 单一静态 token(环境变量 `TALLY_WEBHOOK_TOKEN`),常量时间比较。**
  单用户自部署,不做账号/OAuth(INV-7:不造账号体系)。未配置 token → 路由整体返回 503
  (明确「未启用」,不做默认放行也不做默认拒绝歧义)。
- **D-d2 INV-9 兼容:webhook 挂在登录组之外(token 就是它的登录),但无 token 的请求
  一律非 2xx(401/503)** —— `test/arch/auth_required_test.go` 的行为式检查
  (非登录写请求必须被拒)不需要改就过。不改架构测试。
- **D-d3 CSRF:webhook 不在 CSRF 中间件组内**(非浏览器交互,token 即凭据)。
- **D-d4 原始日志落盘留存**(PRD 第三部分「原始日志留一份,规则改了能重跑」):
  写到 `TALLY_LOG_DIR`(默认 `./tally-logs`)/<项目>/<UTC时刻>-<commit 短哈希>.log,
  `raw_log_ref` 记该路径。单条上限 10MB(PRD 同限),请求体 12MB 封顶。
- **D-d5 录入复用 `ingest.IngestBuild`**,webhook 只是运输层:指纹/事实面/索引面
  语义与 CLI 完全一致;成功构建(status=passed)同样只写 build_run 不录失败。

## 交付物

```
internal/web/webhook.go       POST /webhook/build:token 校验 → 解 JSON → 存日志 → IngestBuild → 回 JSON
internal/web/webhook_test.go  httptest:401(错 token)/ 503(未配置)/ 405(非 POST)/ 413(超尺寸)
internal/web/webhook_integration_test.go  真 Postgres:合法请求落 build_run + failure_event,raw_ref 指向留存文件
internal/web/router.go        注册路由(登录组外);Deps 加 IngestStore/LogDir/WebhookToken
cmd/server/main.go            装配:读 TALLY_WEBHOOK_TOKEN / TALLY_LOG_DIR
docs/07-operations.md         环境变量与 curl 示例
```

请求体(JSON):`project` `commit` `toolchain` `status` `exit_code` `ci_url` `artifact`
`raw_ref`(可空,缺省用留存路径)`combo{k:v}` `workdir` `repro` `other_targets[]` `log`。

## 验收标准(机器可查)

- [ ] 无/错 token → 401;未配置 token → 503;两者都非 2xx(INV-9 行为测试不改而过)
- [ ] 集成:合法 POST → build_run + failure_event 落库,日志文件落盘,raw_log_ref 指向它
- [ ] log 超 10MB → 413 且不录入
- [ ] `make check` 等价全套全绿(含 test/arch 原样通过)

## 不要做

- 不做 CI 平台专属适配(GitHub/GitLab 各自 schema)—— 通用 JSON,CI 侧用一行脚本拼
- 不做重试队列/签名轮换/HMAC(token + 单机自部署够用;复杂化记 BACKLOG)
- 不在 webhook 里做自动诊断入队(那是 T11e,环境变量开关,与本任务正交)

## 相关文档

`docs/00-PRD.md` 第三部分「从哪收」「原始日志留一份」;`docs/01-architecture.md`
(cmd/server 职责含「接收 webhook」—— 本任务把这句兑现);`tasks/T10a-ingest-build.md`(录入语义)。
