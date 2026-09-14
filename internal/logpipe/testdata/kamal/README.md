# kamal 失败样本

| 文件 | 失败 | 为什么留它 |
|---|---|---|
| `deploy-fail.log` | 推镜像失败(registry 鉴权/不存在) | kamal 的 `HH.HHs ERROR` 时间前缀格式;IP 须被洗成 `<ip>` |
| `health-wait-timeout.log` | 健康检查等待超时 | kamal 部署后等健康的典型失败;时间戳前缀须被洗 |

(IP 用 RFC5737 文档段 192.0.2.x,非真实主机;tag/sha 为构造值。)
