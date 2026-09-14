# T01 环境拉起

## 前置
T00

## 目标
读环境定义,把开发/测试环境拉起来,跑健康检查,记录版本组合。

## 交付物

```
internal/runner/            Runner 接口 + DockerRunner + SSHRunner
internal/environment/
  definition.go             环境定义解析(.tally/environments/*.yaml)
  health.go                 健康检查执行
  lifecycle.go              拉起 / 推倒 / 重建
  registry.go               注册校验
migrations/0003_env.sql     environment_run 表
cmd/tally/env.go            手动触发
```

环境定义至少支持:`docker-compose` 与 `ssh-script` 两种;健康检查至少支持 `tcp` `http` `log-pattern` 三种。

## 验收标准

- [ ] 缺少健康检查的定义,注册时返回 `ErrNoHealthCheck`,**拒绝注册**(不是警告)
- [ ] 拉起成功后 `environment_run` 有一行,`version_combo` 非空且包含镜像 tag 与运行时版本
- [ ] `definition_hash` 是定义文件的内容哈希;改定义后新记录哈希不同(INV-4)
- [ ] 拉起失败时把原始输出交给 `logpipe` 的入口(T04 未完成时先落盘,留好接口)
- [ ] 推倒重建后能拉起,不残留卷和网络
- [ ] 集成测试:用 `alpine` 起 sleep 容器,验证拉起 → 健康检查 → 记录 → 推倒全流程
- [ ] `environment_run` 无 UPDATE 路径

## 不要做
- 不碰生产环境
- 不做 k8s
- 不自己实现健康等待重试策略之外的编排

## 相关文档
`docs/00-PRD.md` 第一部分 · `docs/01-architecture.md`
