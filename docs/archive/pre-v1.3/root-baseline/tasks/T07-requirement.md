# T07 需求契约

## 前置
T00

## 目标
需求文件格式、linter 五闸、LLM 问询循环、签署冻结。

## 交付物

```
internal/requirement/
  parse.go        EARS 格式解析(.tally/requirements/*.md)
  lint.go         L1–L5
  interview.go    缺口 → 问题(调 llm)
  sign.go         内容哈希 + git signed commit
  invalidate.go   条款变更 → 下游失效
migrations/0007_clause.sql   clause / clause_version 表
cmd/tally/req.go             lint / interview / sign
```

## 验收标准

### 记法
- [ ] 采用 EARS,**不自创语法**(https://alistairmavin.com/ears/)
- [ ] 解析 WHILE / WHEN / IF-THEN / WHERE 及无关键词的常态需求
- [ ] ruleset 校验:零或多前置、零或一触发、恰好一个系统名、一或多响应

### linter 五闸
- [ ] L1 条款化:每条可独立编号引用
- [ ] L2 验收条件:每条都有,且指明检查机制(自动化测试 / 指标 / 人工检查)
- [ ] L3 零未决:无 TBD、待定、未选的二选一
- [ ] L4 非目标非空:必须显式写"不做什么"
- [ ] L5 术语唯一:同一概念一个词,附术语表
- [ ] 每闸有正反两个测试用例
- [ ] 输出是结构化问题清单(第几条、缺什么),**不是评分**

### 问询
- [ ] LLM 只做"缺口 → 问题"的转换,问题尽量带候选答案
- [ ] LLM **不产出需求正文**(测试:提案的 JSON schema 里没有正文字段)
- [ ] 结构缺口是硬闸,不清零不许签
- [ ] 实质缺口只建议,可一键驳回,**不阻塞签署**
- [ ] 驳回留痕,存在 git

### 签署与失效
- [ ] 签署产出内容哈希 + git signed commit
- [ ] 条款内容变更后,引用旧哈希的下游记录自动显示"依据已变更"(INV-4,不需要额外机制)
- [ ] `clause_version.content_hash` 唯一约束

## 不要做
- 不重造起草循环(Tessl / Spec Kit / Kiro 已有,可消费)
- 不做需求评审流程、工时、指派
- 不让 LLM 判断"这个需求写得够不够好"

## 相关文档
`docs/00-PRD.md` 第六部分
