# T06 失败诊断与验证

## 前置
T04

## 目标
没命中知识库时,LLM 提修复方案,四道闸验证,过了才入库。

## 交付物

```
internal/llm/          Proposer 接口(只有 Propose)+ HTTP 客户端
internal/knowledge/    查库 / 写库 / 老化标记
internal/failure/
  diagnose.go          调 llm 提案
  gates.go             G1–G4
  classify.go          五种出口
  budget.go            三轮上限
knowledge/             git 里的已验证修复
migrations/0006_fp.sql fingerprint_stat 表
```

## 验收标准

### 查库
- [ ] 精确指纹命中 → 直接返回修复,**零 LLM 调用**(测试要断言调用次数为 0)
- [ ] 模糊指纹命中 → 只返回线索;查模糊指纹的方法**签名里不含** `Fix` 字段
- [ ] 命中时比对版本组合,不一致标 `stale` 并降级为线索

### 四道闸
- [ ] G1:连跑 5 次全红才算稳定复现;5 红 4 红 1 绿 → 归类 `flaky`,**不入库**
- [ ] G2:修复后连跑 5 次全绿;4 绿 1 红 → 不入库
- [ ] G3:其余构建目标不受影响
- [ ] G4 白名单:依赖清单、锁文件、Dockerfile、CI 配置、compose/部署清单、资源限制、健康检查定义
- [ ] G4 改到应用源码 → 暂停等人确认
- [ ] G4 **改到验证用的文件 → 直接拒绝**(返回 `ErrFixTouchesTests`),不是暂停

### 入库
- [ ] `knowledge.Store` 签名要求 `VerificationProof`,该类型只能由 `gates` 返回,构造函数不导出
- [ ] 入库动作是 git commit,不是数据库写入
- [ ] 不存 LLM 分析过程、故障描述、未验证结论

### 出口与预算
- [ ] 五种出口全部实现:`fixed` `flaky` `upstream` `infra` `spec_defect`
- [ ] `spec_defect` 回写 `requirement`,触发失效传播(T07 未完成时先留接口)
- [ ] 诊断三轮不过 → 归档"暂无自动修复";同指纹再命中直接返回该结论,**不再调 LLM**
- [ ] 送进 `llm` 的内容必须先过 `logpipe.MustBeRedacted`

## 不要做
- 不做运行期错误分析(那是 Sentry 的事)
- 不做 flaky 治理(只识别并排除,治理用现成工具)
- 不给 LLM 任何判定角色

## 相关文档
`docs/00-PRD.md` 第四、五部分 · `docs/02-invariants.md`
