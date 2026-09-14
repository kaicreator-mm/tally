# go 失败样本

| 文件 | 失败 | 为什么留它 |
|---|---|---|
| `compile-error.log` | `go build` 编译错误(undefined、类型不符) | 检验「保留 行:列 与错误文本」:精确指纹带行号,不同行不同指纹 |
| `vet-fail.log` | `go vet` 静态检查失败 | `FAIL <pkg> [build failed]` 是 go 工具链特征结尾 |
| `repeated-undefined/` | 同一编译错误重复 3 次 | 碰撞测试(go 版):同错误跨运行须撞同一 `fp_exact` |
