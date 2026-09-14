# T03 部署

## 前置
T01

## 目标
把已构建的制品部到环境上。调外部工具,不自己实现部署逻辑。

## 交付物

```
internal/deploy/
  definition.go     部署定义解析
  tool.go           kamal / ansible / compose 三个适配器
  deploy.go         调度 + 记账
  rollback.go       调用工具自带的回滚
migrations/0004_deploy.sql   deploy_run 表
cmd/tally/deploy.go
```

## 验收标准

- [ ] **输入必须是已存在的制品 digest。** 部署定义里出现"拉代码后构建"的形态 → 解析时报错
- [ ] `deploy_run.artifact_digest` 非空约束在数据库层
- [ ] 每次部署写一行:digest、目标、工具、结果、退出码、健康检查结果、版本组合
- [ ] 健康检查与回滚由外部工具执行,本包只读退出码;**代码里不出现等待重试循环**
- [ ] 失败输出送 `logpipe`,`source` 标为 `deploy`
- [ ] 验证用的重新部署跑在隔离环境,不复用正在使用的测试环境
- [ ] `internal/deploy` 代码量 < 500 行(超了说明在自己写部署引擎)

## 不要做
- 不实现零停机切换、健康等待、滚动更新、回滚逻辑
- 不做制品库
- 不碰生产

## 相关文档
`docs/00-PRD.md` 第二部分
