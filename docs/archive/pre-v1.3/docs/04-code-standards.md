# 代码规范

Go 的通用规范以 [Effective Go](https://go.dev/doc/effective_go) 和 [Google Go Style Guide](https://google.github.io/styleguide/go/) 为准,本文只写**这个项目额外要求的**和**容易做错的**。

---

## 依赖白名单

INV-6 的检查从这里读。**加依赖 = 改这个清单**,改动会出现在 diff 里。

```
github.com/jackc/pgx/v5           数据库驱动
github.com/pressly/goose/v3       迁移
github.com/go-chi/chi/v5          HTTP 中间件
github.com/go-git/go-git/v5       读 git
golang.org/x/crypto               SSH
golang.org/x/sync                 errgroup
gopkg.in/yaml.v3                  环境定义 YAML 解析(.tally/environments/*.yaml)
```

工具(不进 go.mod 的 require,用 `tools.go` 管):`sqlc`、`golangci-lint`、`goose` CLI。

**不引入:** ORM、依赖注入框架、日志框架(用 `log/slog`)、断言库、mock 框架、前端构建工具。

---

## 错误处理

### 一律 wrap,带上下文

```go
if err != nil {
    return fmt.Errorf("拉起环境 %s: %w", envName, err)
}
```

上下文写**做什么失败了**,不是"error occurred"。

### 领域错误用哨兵值

```go
var (
    ErrNoHealthCheck   = errors.New("环境定义缺少健康检查")
    ErrNotReproducible = errors.New("失败无法稳定复现")
    ErrFixTouchesTests = errors.New("修复改动了验证用的文件")
)
```

调用方用 `errors.Is` 判断。**不要靠字符串匹配。**

### 不 panic

只有一个例外:`init` 里加载嵌入资源失败可以 panic —— 那种情况二进制本身是坏的。

---

## 命名

### 包名

单个小写词,不用缩写不用下划线。`logpipe` 不是 `log_pipe`,`knowledge` 不是 `kb`。

**不要 `utils` `common` `helpers` `base`。** 想不出名字说明职责没想清楚。

### 类型

领域概念用具名类型,不用裸 string:

```go
type ProjectID string
type Fingerprint string
type ArtifactDigest string
type ClauseHash string
```

这几个混起来是必然会发生的错误,类型系统能免费挡掉。

### 函数

动词开头。返回 bool 的用 `Is` / `Has` / `Can`。

**不要用 `Check` 和 `Validate` 作为返回错误的函数名** —— 容易和 LLM 判定混淆。要么 `Verify`(有确定性判据、返回证据),要么 `Lint`(返回结构化问题清单)。

---

## 接口

### 在使用方定义,不在实现方

```go
// internal/environment/environment.go —— 使用方
type Runner interface {
    Run(ctx context.Context, spec RunSpec) (RunResult, error)
}
```

`runner` 包提供实现,不导出接口。

### 保持小

超过三个方法就该想想是不是两个概念。

### `llm.Proposer` 只能有 `Propose`

INV-1 会检查。要加方法 —— 先问这个方法会不会让 LLM 变成判定者。

---

## 并发

### `context.Context` 是第一个参数,一路传下去

不存在结构体里。

### 不要裸起 goroutine

用 `errgroup`,或者 `jobs` 包的 worker。

**没有 owner 的 goroutine 会在关机时留下半截状态**,而这个工具的半截状态就是脏数据。

### worker 的并发度可配

默认 2。拉环境和跑构建都是重活,并发高了机器扛不住。

---

## 数据库

### 手写 SQL,`sqlc` 生成

不用 ORM,不拼字符串。

SQL 放 `internal/store/queries/*.sql`,按表分文件。

### 事务边界在业务包,不在 store

```go
// 业务包
err := store.InTx(ctx, func(q *store.Queries) error {
    ...
})
```

### 事实面表只有 Insert 和 Select

`sqlc` 查询文件里不写这些表的 UPDATE/DELETE。INV-5 会检查。

### 时间一律 `TIMESTAMPTZ`,一律 UTC

显示时再转。

---

## 日志

用 `log/slog`,结构化。

```go
slog.Info("环境拉起完成",
    "project", projectID,
    "env", envName,
    "duration", d,
)
```

**不要在日志里打原始日志内容。** 那些东西可能带密钥,而且已经存在数据库里了。打引用(`failure_event.id`)。

级别:

- `Error` —— 需要人看的
- `Warn` —— 降级了但还能跑(比如修复标记为 stale)
- `Info` —— 状态变化(拉起、部署、入库)
- `Debug` —— 流水线各阶段的中间结果

---

## 脱敏

**这是安全边界,单独说。**

- 所有要送进 `llm` 包的内容,必须先过 `logpipe.Redact`
- `logpipe` 提供一个 `MustBeRedacted(s string) error`,在 `llm` 包入口调用,发现疑似密钥模式就拒绝发送
- 脱敏规则本身要有单元测试,用构造的假密钥当样本

**宁可误伤。** 把一个正常字符串洗掉了,最多是诊断质量下降;漏一个 token,它就同时躺在数据库、git 和别人的服务器上。

---

## 文件与包组织

### 一个包一个 `doc.go`

写清楚:这个包干什么、不干什么、允许导入谁。

### 文件按职责切,不按类型切

`environment/health.go` `environment/definition.go`,不是 `environment/types.go` `environment/funcs.go`。

### 单文件超过 500 行就拆

不是硬规矩,是信号。

---

## 注释

### 导出的东西必须有注释,以名字开头

```go
// Verify 跑完四道闸,全部通过时返回证据。
// 任何一闸不过返回对应的哨兵错误,证据为零值。
func Verify(ctx context.Context, c Candidate) (VerificationProof, error)
```

### 注释写"为什么",不写"什么"

代码已经说了什么。注释要说的是当初为什么这么选。

```go
// 连跑 5 次而不是 1 次:单次红绿可能是偶发,
// 偶发故障以"已验证修复"身份入库会污染知识库(见 docs/02-invariants.md INV-2)
const reproRuns = 5
```

### 中文注释可以,但术语用原文

`fingerprint` 不写"指纹哈希值",`digest` 不写"摘要"。文档里可以用中文,代码里保持和字段名一致。

---

## 格式与检查

`gofmt` 强制。`golangci-lint` 配置在 `.golangci.yml`,启用:

```
errcheck govet staticcheck unused gosimple ineffassign
bodyclose contextcheck errorlint nilerr
```

`errorlint` 特别重要 —— 它会抓到 `err == ErrFoo` 这种漏掉 `errors.Is` 的写法。
