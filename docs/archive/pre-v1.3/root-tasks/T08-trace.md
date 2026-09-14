# T08 追溯挂钩与两个视图

## 前置
T06 T07

## 目标
把条款挂到提交上,打通追溯链,做出视图 A 和视图 B。

## 交付物

```
internal/project/
  trailer.go     解析 commit trailer `Refs: REQ-12`
  annotation.go  解析测试注解 `@covers REQ-12.2`
  trace.go       追溯图构建
  views.go       视图 A / 视图 B 查询
migrations/0008_trace.sql   commit_clause / test_clause / trace_edge
cmd/tally/reindex.go        全量重算
.github/ 或 CI 配置          trailer 存在性检查(required status check)
```

## 验收标准

### 挂钩
- [ ] 从 commit trailer 解析条款引用,按**当时的当前版本**绑定内容哈希
- [ ] 从测试注解解析条款引用,产出条款覆盖率
- [ ] CI 检查:提交缺 `Refs:` 时阻断合并(作为 required status check,不是服务端 hook)

### 追溯链
- [ ] 链路可查:条款哈希 → commit → 制品 → 部署 → 失败指纹
- [ ] 全部从机器记录推导,**无人工标注字段**

### 视图 A(某条款的实现累计失败)
- [ ] 显示:实现提交数、构建失败数、部署失败数、其中同一指纹的次数、条款年龄与变更次数
- [ ] 失败构成按指纹列出,标注处置分类

### 视图 B(某指纹的影响面)
- [ ] 显示:出现次数、影响项目数、关联条款、已验证修复及其验证时组合、是否漂移
- [ ] **关联条款用 LEFT JOIN**:没有需求层时该列为空,页面其余部分照常工作

### 重建
- [ ] `make reindex` 可 drop 全部索引面表并从 git + 事实面重算
- [ ] CI 定期跑:重建后与现有数据 diff,不一致即失败(说明有人绕过了单向数据流)

## 不要做
- 不做手动挂钩界面(挂钩来自 trailer 和注解,不来自人点)
- 不做任务状态字段(每次算)

## 相关文档
`docs/00-PRD.md` 第六部分「和前面几部分怎么连」· `docs/03-data-model.md`
