# compose 失败样本

| 文件 | 失败 | 为什么留它 |
|---|---|---|
| `healthcheck-timeout.log` | 服务健康检查失败(连接被拒) | 每行带 `svc_1  \|` 前缀,检验**并行日志按服务分组**后只抽失败组 |
| `container-exit.log` | 容器内 Python 进程崩溃 | 前缀日志 + Python `Traceback`,检验分组后**堆栈整段取** |
