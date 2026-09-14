# docker 失败样本

| 文件 | 失败 | 为什么留它 |
|---|---|---|
| `build-fail.log` | buildkit 构建步骤失败(容器内 go build 报错) | `did not complete successfully: exit code: 1` 是 buildkit 特征;保留退出码 |
| `pull-fail.log` | 拉基础镜像失败(网络/鉴权) | `load metadata for ... ` + `pull access denied` 是拉镜像失败特征 |
