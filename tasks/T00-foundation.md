# T00 地基

## 前置
无。

## 目标
把项目跑起来:能编译、能连库、能跑迁移、架构测试就位、**登录/CSRF 地基就位**。

## 交付物

```
go.mod  go.sum                    依赖限定在白名单内
Makefile                          build / test / check / migrate / reindex
.golangci.yml
docker-compose.dev.yml            本地 Postgres
internal/core/                    ProjectID Fingerprint ArtifactDigest ClauseHash VersionCombo
internal/store/                   sqlc 配置 + 连接 + InTx
internal/jobs/                    job 表 + worker pool(并发默认 2)
migrations/0001_facts.sql         事实面五张表 + 阻止 UPDATE/DELETE 的触发器
migrations/0002_index.sql         索引面表
migrations/0009_session.sql       session 表(登录态)
internal/web/                     路由 + 登录/CSRF 中间件 + 登录/登出 + 健康端点(页面 handler 归 T05a)
cmd/server/  cmd/worker/  cmd/tally/   只有骨架和健康端点
test/arch/                        不变量测试(INV-1..8 + INV-9 写路由过登录)
```

## 验收标准

- [ ] `make check` 全绿
- [ ] `test/arch/imports_test.go` 存在且通过:`logpipe`/`knowledge`/`store` 不导入 `llm`;`core` 不导入任何 internal 包
- [ ] `test/arch/append_only_test.go` 通过:事实面表无 UPDATE/DELETE 查询
- [ ] `test/arch/fk_hash_test.go` 通过:引用条款/组合/制品的外键列名以 `_hash` 或 `_digest` 结尾
- [ ] `test/arch/no_manual_fields_test.go` 通过:黑名单字段不存在
- [ ] `test/arch/deps_test.go` 通过:`go.mod` 直接依赖 ⊆ `docs/04-code-standards.md` 白名单
- [ ] `test/arch/auth_required_test.go` 存在且通过:非 GET/HEAD 路由全挂在登录中间件后;公开路由仅登录/登出/健康/静态(INV-9)
- [ ] 未登录访问非公开路由重定向 `/login`;密码用 `x/crypto/bcrypt`;`go.mod` 无新依赖
- [ ] 手工验证:直接对事实面表执行 `UPDATE` 被数据库拒绝
- [ ] `make migrate` 可重复执行,幂等
- [ ] 三个 cmd 都能启动并响应健康检查

## 不要做
- 不实现任何业务逻辑
- 不写界面
- 不建索引(除 `docs/03-data-model.md` 里列的三个)

## 相关文档
`docs/02-invariants.md` · `docs/03-data-model.md` · `docs/04-code-standards.md`
