# npm 失败样本

| 文件 | 失败 | 为什么留它 |
|---|---|---|
| `eresolve.log` | 依赖冲突 ERESOLVE(peer dependency 不兼容) | npm 最常见的构建失败;`code ERESOLVE` 是识别标记 |
| `lockfile-mismatch.log` | `npm ci` 锁文件与 package.json 不同步 | CI 环境高发;检验「保留包名与版本号」不被误洗 |
| `native-module-abi/` | 原生模块 ABI 不匹配(3 份) | 碰撞测试:同错误多份须撞同一 `fp_exact` |
