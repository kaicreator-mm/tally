# T02 服务定义抽取

## 前置
T01(需要拉起来验证)

## 目标
从 Coolify 等来源抽服务定义,实际拉起验证,通过的存进 `catalog/`。

## ⚠ 开工前的硬节点

**许可证要先查完。** Coolify 的服务定义多为社区贡献的 compose 文件,衍生使用的许可与署名要求需单独确认。

结论写进 `catalog/LICENSE-NOTES.md`,**没有这个文件不允许开始抽取**。

## 交付物

```
internal/harvest/
  source.go       来源适配(coolify / 上游 compose 示例 / 官方文档)
  propose.go      调 llm 产出定义 + 健康检查
  verify.go       调 environment 实际拉起
  catalog.go      写 catalog/ 并 commit
catalog/
  LICENSE-NOTES.md
  <service>/definition.yaml + verified.json
cmd/tally/harvest.go
```

## 验收标准

- [ ] LLM 产出必须包含健康检查;不含则视为提案失败,不进验证
- [ ] 只有实际拉起并通过健康检查的定义才写入 `catalog/`(INV-2)
- [ ] 写入函数签名要求 `VerificationProof`,无法绕过
- [ ] `verified.json` 含验证时间与完整版本组合
- [ ] 三轮不通过则跳过该服务并记录原因,不无限重试
- [ ] 抽取清单可配置,默认十来个;**不做全量批处理**
- [ ] `internal/harvest` 是 `llm` 的合法调用方之一(架构测试白名单里)

## 不要做
- 不批量抽 200+ 个服务
- 不 fork Coolify,不依赖它的运行时
- 不存"抽出来但没验证"的定义

## 相关文档
`docs/00-PRD.md` 第一部分「环境定义从哪来」
