# 不变量与机器检查

七条铁律。每条都配一个自动检查 —— **写在文档里的规矩会被忘,写成测试的不会。**

全部由 `make check` 执行。

> **v3(平台化)说明:** 七条铁律**不变**。变的是:去掉「界面纯只读」这条 UI 级过严约束;澄清 INV-3、INV-7 的边界(限状态/跟踪字段,不限用户内容);新增 INV-9(写路由必须过登录)。

---

## INV-1 LLM 只能提案

**检查:** `test/arch/imports_test.go`

解析全部包的导入图,断言:

```
internal/logpipe    不导入 internal/llm
internal/knowledge  不导入 internal/llm
internal/store      不导入 internal/llm
internal/core       不导入任何 internal 包
```

**辅助检查:** `test/arch/llm_interface_test.go`

用反射断言 `llm` 包导出的接口只有 `Propose` 这一个方法。出现 `Judge` `Verify` `Score` `Evaluate` `Check` `Validate` 之类的方法名 → 失败。

> 接口形状是最便宜的约束。方法不存在,就没人能调错。

---

## INV-2 未验证的不入库

**检查:** `internal/knowledge` 和 `internal/harvest` 的写入函数签名里必须带验证证据。

```
// 允许
Store(ctx, fp Fingerprint, fix Fix, proof VerificationProof) error

// 不允许
Store(ctx, fp Fingerprint, fix Fix) error
```

`VerificationProof` 是不可伪造的:只能由 `failure` 的闸门函数或 `environment` 的健康检查函数返回,构造函数不导出。

**测试:** `knowledge` 包外无法构造 `VerificationProof`(编译期保证 + 一个 `go vet` 风格检查)。

**机器检查:** `test/arch/proof_guard_test.go`

- failure 里实现验证证据的类型(带 `VerifiedAt/VerifiedCombo/GateRecord/Valid` 方法)字段必须**全部不导出** —— 包外构造不出带内容的证据;
- knowledge 的写库方法签名必须带 `VerificationProof` 参数。

---

## INV-3 状态不给手动改

**检查:** `test/arch/no_status_update_test.go`

扫描 `store` 的生成代码和 `web` 的路由:

- `sqlc` 生成的查询里,不允许存在对状态字段的 `UPDATE`
- HTTP 路由里不允许出现形如 `PUT/PATCH /tasks/{id}/status` 的路径

状态字段清单在 `docs/03-data-model.md`,测试从那里读。

**v3 边界(重要):** 禁的是改**状态字段**和**状态路由**。平台对用户内容的增删改(建项目、编辑需求正文、写 Wiki 笔记、记 bug 备注)是合法写操作 —— 它们写 git 或产生**新的**事实记录,**不碰状态字段**。测试只断言上面两条,不能误伤内容 CRUD 路由。判断标准:这个写操作最终是改了某个 `.status`,还是新写了一条记录/一个 git 文件?前者禁,后者放行。

---

## INV-4 挂钩记内容哈希

**检查:** `test/arch/fk_hash_test.go`

扫描迁移文件,断言:任何引用条款、版本组合、制品的外键列,列名以 `_hash` 或 `_digest` 结尾,且类型是定长字符。

出现 `clause_id INTEGER REFERENCES clause(id)` 这类 → 失败。

> 只记 ID 的话,条款改了图还连着,看起来一切正常,实际全是过期的 —— 而且不会报错。

---

## INV-5 事实面只追加

**检查:** `test/arch/append_only_test.go`

事实面表清单(在 `docs/03-data-model.md` 里标记),断言:

- `sqlc` 查询里没有针对这些表的 `UPDATE` / `DELETE`
- 迁移里这些表有数据库级触发器阻止 UPDATE/DELETE(双保险)

---

## INV-6 不写别人写好的东西

**检查:** 半自动。`test/arch/deps_test.go` 断言 `go.mod` 的直接依赖在白名单内。

白名单在 `docs/04-code-standards.md`。要加依赖 → 改白名单 → 这个改动会出现在 diff 里被看见。

不能完全自动检查的部分(比如"有没有偷偷自己写了个部署引擎")靠代码审查。**信号:`deploy` 包超过 500 行就该警觉。**

---

## INV-7 只做有机器记录对应的功能

**检查:** `test/arch/no_manual_fields_test.go`

字段黑名单:`estimated_hours` `actual_hours` `assignee` `priority` `story_points` `reviewer` `approver` `due_date`。

迁移文件里出现这些列名 → 失败。

> 这个清单不是穷举,是把最容易长回来的那几个钉死。

**v3 澄清:** 这条限的是*机器填不出的跟踪字段*,**不限用户创作的内容**。需求正文、文档、Wiki 笔记是人写的(存 git,界面可编辑),不属于「字段」,不受此条限制。

---

## INV-8 logpipe 是纯函数

不在七条铁律里,但同样重要。

**检查:** `internal/logpipe` 的每个阶段函数满足:

- 无包级变量
- 不读时钟、不读随机数、不读环境变量
- 相同输入必然相同输出

**测试:** 每个阶段用固定输入跑两次,断言结果 byte 级相同。

理由:指纹必须可复现。任何不确定性混进来,同一个错误就会算出不同指纹,而这个故障是静默的。

---

## INV-9 写路由必须过登录(v3 新增)

不在七条铁律里,但平台化之后必须有。

**检查:** `test/arch/auth_required_test.go`

枚举所有注册的 HTTP 路由,断言:

- 任何**产生副作用的路由**(GET/HEAD 以外的方法)都挂在登录中间件之后
- 公开的只有:`/login`、`/logout`、健康检查、静态资源
- 写路由同时带 CSRF 防护(双提交 token,手写,不引依赖)

> 平台有了写操作,就不能让它们裸露在网络上。**单用户 ≠ 无鉴权。** 简单登录(密码哈希 `x/crypto/bcrypt` + session 表)是下限。

---

## `make check` 都跑什么

```
make check
  ├── go build ./...
  ├── gofmt -l          (有输出即失败)
  ├── go vet ./...
  ├── golangci-lint run
  ├── go test ./test/arch/...     ← 上面这些不变量
  ├── go test ./internal/...      ← 单元测试
  └── go test -tags=integration ./...  ← 需要 Postgres
```

**架构测试跑在单元测试前面。** 结构错了,功能测试全绿也没意义。

---

## 加不变量的规矩

发现一类反复出现的错误 → 先写检查,再修代码。

不要只修这一次的问题。**"下次注意"不是机制。**

新增的检查放 `test/arch/`,并在本文件登记一条。
