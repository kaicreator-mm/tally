# 测试规范

## 三层

| 层 | 位置 | 跑什么 | 要不要外部依赖 |
|---|---|---|---|
| **架构测试** | `test/arch/` | 七条铁律(见 `docs/02-invariants.md`) | 不要 |
| **单元测试** | `internal/*/xxx_test.go` | 纯逻辑 | 不要 |
| **集成测试** | `internal/*/xxx_integration_test.go`,`//go:build integration` | 数据库、docker、git | 要 |

**架构测试先跑。** 结构错了,功能全绿也没意义。

---

## 先写测试

不是教条,是因为这个项目里**大部分"能不能算完成"本身就是二值判据**,写测试就是把判据写下来。

尤其这三处必须先写:

- `logpipe` 各阶段 —— 输入输出都是确定的
- `failure` 四道闸 —— 闸门逻辑就是判据本身
- `requirement` linter 五闸 —— 同上

---

## 单元测试

### 表驱动

```go
tests := []struct{
    name string
    in   string
    want string
}{
    {"去掉 ANSI 颜色码", "\x1b[31merror\x1b[0m", "error"},
    {"保留退出码", "exit status 137", "exit status 137"},
}
```

子测试名用中文可以,要说清楚测的是哪个行为,不要 `case1` `case2`。

### 不用断言库

`if got != want { t.Errorf(...) }` 就够。失败信息要包含实际值和期望值。

### 不 mock 内部包

需要替身时用接口 + 手写的假实现,放 `xxx_fake.go`。

**不引入 mock 生成框架。** 这个项目的接口都很小,手写更清楚。

---

## `logpipe` 的测试要求特殊

这个包决定整个知识库有没有用,所以要求高一档。

### 用真实日志样本

`internal/logpipe/testdata/` 下按工具链放真实日志:

```
testdata/
  npm/            依赖冲突、原生模块编译失败、锁文件不一致
  go/             编译错误、vet 失败
  docker/         构建失败、拉镜像失败
  compose/        容器起不来、健康检查超时
  kamal/          部署失败、健康等待超时
```

**每个样本要脱敏后再提交。** 样本本身就可能带密钥。

### 每个阶段必测三件事

1. **确定性** —— 同一输入跑两次,结果 byte 级相同(INV-8)
2. **该洗的洗掉了** —— 路径、时间戳、PID、地址、临时目录名
3. **不该洗的没洗** —— 退出码、包名、版本号、异常类名

### 指纹的关键测试:同类必须撞

拿同一个错误在不同时间、不同机器、不同路径下的多份日志,断言**算出同一个精确指纹**。

```
testdata/npm/native-module-abi/
  run-1.log   ← 不同时间戳、不同 pid、不同临时目录
  run-2.log
  run-3.log
  expected-fp.txt
```

**这是整个测试套里最重要的一组。** 它们不撞,知识库就永远查不到东西,而且不会报错 —— 只是表现为"这些错误好像各不相同"。

### 反向测试:不同类不能撞

不同根因的失败必须算出不同指纹。洗过头会导致所有错误撞成一个,同样是静默故障。

### 脱敏测试

用构造的假密钥(`ghp_` 开头、`sk-` 开头、JWT、连接串)当样本,断言全部被洗掉。

**这个测试不能只测正例。** 也要测"正常内容没被误伤到影响指纹"。

---

## `failure` 四道闸的测试

每道闸单独测,再测组合。

必测的边界情况:

| 场景 | 期望 |
|---|---|
| 5 次里 4 次红 1 次绿 | G1 不过 → 归类 `flaky`,**不入库** |
| 修复后 5 次里 4 次绿 1 次红 | G2 不过 → 不入库 |
| 修复改了 `Dockerfile` | G4 过 |
| 修复改了 `internal/foo/bar.go` | G4 暂停,等人确认 |
| **修复改了测试文件** | G4 **直接拒绝**,不是暂停 |
| 诊断到第 4 轮 | 拒绝,归档为"暂无自动修复" |

最后两行是防作弊和防烧钱的,必须有测试钉住。

---

## 集成测试

### 用真实 Postgres,不用 sqlite,不用 mock

```go
//go:build integration
```

`make test-integration` 起一个临时 Postgres 容器,跑完销毁。

### 每个测试自己建 schema,自己清理

用独立 schema 而不是独立数据库,快。**统一走 `store.TestDB(t)`**(`internal/store/testdb.go`):
它在 `tally` 库里为本次测试建一个 `tally_it_<纳秒>` schema、把全部迁移(含 INV-5 触发器)
跑进去、返回 `search_path` 已指到该 schema 的 `*store.Store`,`t.Cleanup` 里 `DROP SCHEMA CASCADE`。

```go
func openStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t)   // 独立 schema;不要再用 store.Open + DATABASE_URL 共享默认 schema
}
```

因为每个测试独占一份 schema,**断言可以断全表**(数 `deploy_run`/`job` 全表行数、
`VerifyIndex` 全库 diff),不必担心 `go test -tags=integration ./...` 各包并行时互相污染。
项目名/指纹仍建议用每次运行唯一值 —— 那是为了在同一 schema 内锁定本测试种的那行,不是防并行污染。

### 环境和部署的集成测试用最小镜像

拉 `alpine` 起一个 sleep 容器就够验证流程。**不要在测试里拉 Postgres 镜像去验证环境拉起** —— 慢,而且和被测逻辑无关。

### 集成测试不进默认 `make test`

它们慢,而且要求 docker。`make check` 会跑,但本地快速迭代时用 `make test` 跳过。

---

## 覆盖率

**不设百分比目标。** 覆盖率高不代表测得对。

改成三条硬要求:

1. `logpipe` 每个阶段函数都有确定性测试
2. `failure` 每道闸都有正反两个用例
3. `requirement` linter 每一闸都有正反两个用例

这三处覆盖不全就不算完成,其他地方按判断来。

---

## 测试数据

### `testdata/` 里的东西要能解释

每个样本目录放一个 `README.md`,一句话说清楚这是什么失败、为什么留它。

三个月后你不会记得 `run-3.log` 是干嘛的。

### 不提交大文件

单个样本超过 100KB 就截断。测试要的是特征,不是全文。

---

## 什么时候不写测试

- `cmd/` 下的接线代码
- 纯粹的结构体定义
- 一次性的迁移脚本

**这三类之外,都写。**
