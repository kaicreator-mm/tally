# 架构测试

七条铁律的机器检查。**跑在单元测试前面** —— 结构错了,功能全绿也没意义。

| 文件 | 检查 | 铁律 |
|---|---|---|
| `imports_test.go` | `logpipe`/`knowledge`/`store` 不导入 `llm`;`core` 不导入任何 internal 包 | INV-1 |
| `llm_interface_test.go` | `llm` 导出的接口只有 `Propose` 一个方法 | INV-1 |
| `proof_test.go` | 入库函数签名要求 `VerificationProof`,且该类型不可在包外构造 | INV-2 |
| `no_status_update_test.go` | 无状态字段的 UPDATE 查询;无改状态的 HTTP 路由 | INV-3 |
| `fk_hash_test.go` | 引用条款/组合/制品的外键列名以 `_hash` 或 `_digest` 结尾 | INV-4 |
| `append_only_test.go` | 事实面表无 UPDATE / DELETE | INV-5 |
| `deps_test.go` | `go.mod` 直接依赖 ⊆ 白名单 | INV-6 |
| `no_manual_fields_test.go` | 迁移里不出现字段黑名单 | INV-7 |
| `purity_test.go` | `logpipe` 各阶段同输入两次结果 byte 级相同 | INV-8 |

## 数据来源

这些测试从文档里读清单,不要在测试里硬编码:

- 依赖白名单 → `docs/04-code-standards.md`
- 状态字段清单、字段黑名单、事实面表清单 → `docs/03-data-model.md`

**文档和检查绑在一起,改一个必须改另一个。**

## 不要改这些测试来让别的测试过

发现测试挡路时,先想清楚是不是真的要违反那条铁律。要违反的话,改的是 `docs/02-invariants.md` 和整个设计,不是偷偷改测试。

## 加新的

发现一类反复出现的错误 → 先写检查,再修代码。「下次注意」不是机制。

新增后在本文件和 `docs/02-invariants.md` 各登记一条。
