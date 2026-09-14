# T04 日志流水线

## 前置
T00

## 目标
把失败日志变成稳定的指纹。**全程不用 LLM。**

> 这是整个项目里最重要的一个任务。做砸了不会报错,只会表现为「这些错误好像各不相同」,
> 然后你会得出「知识库没用」的错误结论。
>
> 它同时就是那个一直没跑的碰撞率实验 —— 先只做统计,不做修复。

## 交付物

```
internal/logpipe/
  collect.go        从 CI API / runner 输出 / 容器日志取原始日志
  redact.go         脱敏(粗 + 细两遍)
  extract.go        去 ANSI → 定位标记 → 取片段
  normalize.go      洗掉路径/时间戳/PID/地址/临时名
  fingerprint.go    精确指纹 + 模糊指纹
  version.go        规则版本号,参与指纹计算
  toolchain/        npm go docker compose kamal 各一个抽取器
  testdata/         真实日志样本(提交前必须脱敏)
migrations/0005_failure.sql   failure_event 表
cmd/tally/scan.go             批量扫历史日志,输出碰撞率
```

## 验收标准

### 结构
- [ ] `internal/logpipe` 不导入 `internal/llm`(架构测试)
- [ ] 每个阶段是纯函数:无包级变量、不读时钟/随机数/环境变量
- [ ] 同一输入连跑两次结果 byte 级相同(INV-8)
- [ ] `fp_exact` 与 `fp_fuzzy` 是两个独立列,已验证修复的外键只能指向 `fp_exact`

### 抽取
- [ ] ANSI 颜色码与控制字符在任何其他处理之前去掉
- [ ] 堆栈跟踪整段取,不按行切
- [ ] 并行构建日志先按 step 分组再抽取
- [ ] 单行超长先截断
- [ ] 每个抽取器有基于真实样本的测试

### 规范化
- [ ] 洗掉:绝对路径、时间戳、耗时、PID/线程/端口、内存地址、UUID、内容哈希、临时目录名
- [ ] **保留:退出码、包名与版本号、错误类型/异常类名**
- [ ] 正向测试:同一错误的多份日志(不同时间/机器/路径)算出**同一个** `fp_exact`
- [ ] 反向测试:不同根因的失败算出**不同** `fp_exact`

### 脱敏
- [ ] 覆盖 `ghp_` / `sk-` / JWT / 长 base64 / `Authorization:` / `password=` / `token=` / `secret=` / `user:pass@`
- [ ] 粗筛在落盘前,细筛在抽取后
- [ ] `MustBeRedacted` 导出,供 `llm` 包入口调用
- [ ] 用构造的假密钥做正例测试,并测试正常内容未被误伤

### 版本化
- [ ] 规则版本号参与指纹计算
- [ ] `cmd/tally reindex-fingerprints` 能用留存的原始日志重算历史指纹

### 实验产出
- [ ] `cmd/tally scan` 跑完历史构建/部署失败,输出:总数、去重后指纹数、**碰撞率**、各工具链分布
- [ ] 结果写进 `docs/EXPERIMENT-collision.md`

## 不要做
- 不做 LLM 诊断(那是 T06)
- 不装 Loki / ELK,不做常驻日志管道
- 不收集成功的日志

## 相关文档
`docs/00-PRD.md` 第三部分 · `docs/05-test-standards.md`
