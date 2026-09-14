# testdata —— 日志流水线的真实样本

按工具链分目录。每个目录一个 `README.md` 说明样本是什么失败、为什么留它
(docs/05-test-standards.md:三个月后你不会记得 run-3.log 是干嘛的)。

```
npm/       依赖冲突、锁文件不一致、原生模块 ABI 不匹配(碰撞测试)
go/        编译错误、vet 失败
docker/    构建失败、拉镜像失败
compose/   健康检查超时、容器起不来(并行分组 + Python 堆栈)
kamal/     推镜像失败、健康等待超时
```

**安全:** 所有样本提交前已脱敏。`redact_test.go` 的 `TestTestdataIsRedacted`
会遍历本目录,任何文件命中 `MustBeRedacted` 的密钥模式即测试失败 ——
防止以后有人把带密钥的日志直接丢进来。样本里若出现疑似密钥,一律是构造的假值。

**最重要的测试:** `npm/native-module-abi/` 的三份日志是同一个错误,
必须算出同一个 `fp_exact`(`fingerprint_test.go` 钉住)。它们不撞,
知识库就永远查不到东西,而且这个故障是静默的。
