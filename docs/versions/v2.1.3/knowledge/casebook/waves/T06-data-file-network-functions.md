# T06 · 数据、文件与网络功能抽象

[#567](https://github.com/kaicreator-mm/tally/issues/567)。6 个新案例/6 个新登记来源文件，涵盖下载、同步/备份、事务、导入导出、搜索和协议/流六族。复用 T05/T04 已登记同 Blob 与现有案例对照，不重复来源或案例数量。实际源观察与迁移方案区分；验证建议 NOT_RUN，不复制项目代码。

## T06-C01 · 断点续传：偏移、目标版本与恢复元数据

**分类：** 下载 / L3–L5。**知识：** K03.F3.2。**来源：** T06-S01、T06-S02。**关系：** 机制关联；恢复身份与进度合同。验证意图另见正文，NOT_RUN。

- **实际观察：** libcurl 文档定义 resume offset，FTP 上传的恢复/追加有特定含义；aria2 控制文件按版本解释字节序，在适用 BitTorrent 条件下检查 InfoHash，并拒绝未知版本/非法分片长度。
- **职责及合同：** 传输 offset 不是已验证数据身份；任务记录源身份、已确认进度及临时输出，协议 adapter 定义真正支持的恢复语义。不同协议不能强制相同 resume 保证。
- **对照、反转条件及验证建议（NOT_RUN）：** 对照 DE-001：单次流下载→无恢复状态；同一可确认源的线性续传→offset；多分片可恢复→版本化进度/分片状态。验证目标变化、旧进度文件、未知格式、损坏分片，不能只检查最终下载字节数。

## T06-C02 · 同步与备份：可变副本和历史快照

**分类：** 同步/备份 / L2–L4。**知识：** K02.F3.3。**来源：** T06-S03、T06-S04。**关系：** 机制关联；当前副本和历史快照数据模型。验证意图另见正文，NOT_RUN。

- **实际观察：** Syncthing folder 持有 FolderConfiguration、FileSet、ignore matcher、I/O limiter；Serve 负责运行计数、停止计时器和空闲状态。restic Snapshot 包含时间、parent/tree ID、paths 和排除项。
- **职责及合同：** 同步服务围绕当前集合和持续运行，备份模型围绕指定时点的历史树；共同的数据移动机制不代表删除、冲突、恢复策略相同。这里不推断 Syncthing 完整同步算法或 restic 一定是原子快照。
- **对照、反转条件及验证建议（NOT_RUN）：** 小目录单向拷贝、双向持续同步、版本化备份是不同产品合同；对照 DE-002。验证源删除传播、同时修改、旧版本恢复、排除规则和中断，先明确是否允许丢弃历史。

## T06-C03 · 数据库事务：连接绑定和结束状态

**分类：** 事务 / L3–L5。**知识：** K03.F4.1。**来源：** T05-S08。**关系：** 机制关联；连接绑定的事务句柄。验证意图另见正文，NOT_RUN。

- **实际观察：** Go database/sql BeginTx 选择连接/事务，非默认隔离级别驱动不支持则报错；Commit 入口先检查 context，再以 CAS 防止已结束 Tx 重复结束。该源已登记 T05-S08（L487–498,1861–1873,2279–2294），本案使用同 Blob。
- **职责及合同：** 池负责连接；Tx 是绑定资源的事务句柄；上下文取消、事务已结束与真实外部提交结果应分别确认。不能从 done 标志推断未知远端效果。
- **对照、反转条件及验证建议（NOT_RUN）：** 对照 AB-014 Django Atomic/savepoint 与 T04-C05：单事务 scope、嵌套 savepoint、跨外部系统协调不同。验证不支持隔离、重复 Commit、取消竞争和 lost ACK；本案未审查全部 Commit 后续实现。

## T06-C04 · CSV 导入导出：字段映射与缺失/额外列

**分类：** 数据导入导出 / L4–L5。**知识：** K03.F2.2。**来源：** T06-S05。**关系：** 机制关联；字段结构映射与缺失额外策略。验证意图另见正文，NOT_RUN。

- **实际观察：** DictReader 接受 fieldnames/restkey/restval/dialect；DictWriter 接受 fields/restval/extrasaction，拒绝非 raise/ignore 的额外字段策略。
- **职责及合同：** 格式 reader/writer、列映射、业务类型校验、导入提交/进度分开；缺列、额外列、空字符串、null、未授权字段不能默认为同义。
- **对照、反转条件及验证建议（NOT_RUN）：** 小表直接读取验证；大批次流式导入+可追踪错误行；对照 DE-003 与 T05-C04/05。验证引号/分隔符、长短行、重复列、额外敏感字段、错误行重试；CSV 解析成功不表示业务提交。

## T06-C05 · 搜索：命中集、排名、分页与权限

**分类：** 搜索 / L3–L5。**知识：** K02.F2.2。**来源：** T04-S03、T04-S04。**关系：** 机制关联；结果范围与截断合同。验证意图另见正文，NOT_RUN。

- **实际观察：** 重用并实际重新读取 T04-S04 Lucene IndexSearcher（L524–558）；T04-S03 Meilisearch 源已在既有 T04 中登记，实际入口有 tenant filter、许可和索引选择。
- **职责及合同：** 搜索 API 分别表达匹配/过滤范围、排序与 top-k、分页身份、部分结果。不能将权限条件只当排名因素，也不能从 top-n 推断全集。禁止越权文档被返回、披露或占据授权 top-k；共享统计改变合法排序与侧信道是另一项须明确授权的强非干扰合同。k=0、超时或部分结果为空不等于授权全集空。
- **对照、反转条件及验证建议（NOT_RUN）：** 对照 T04-C03/C04 和 DE-007：有限数据直接扫描，复杂全文用库，独立多租户服务另加资源/权限边界；验证并列排序、旧索引、截断、权限变化与空结果。未重跑查询或读取完整排序/分页实现。

## T06-C06 · HTTP 运输帧与流资源的结束条件

**分类：** 协议/流 / L3–L5。**知识：** K03.F4.2。**来源：** T06-S06、T05-S10。**关系：** 机制关联；流资源关闭和运输边界。验证意图另见正文，NOT_RUN。

- **实际观察：** Go transferWriter 区分 ContentLength=-1 未知与 0 无内容，Body/BodyCloser/encoding/trailer；writeBody 的 defer 在未关闭时关闭 body，并在主错误为空时保留 close error。Go io.Copy 可选 ReaderFrom/WriterTo 快路径。
- **职责及合同：** 流数据、协议 framing、资源关闭和业务提交各有合同；兼容未知长度，不用 EOF/关闭替代业务完整性。适配器封装协议而不是所有调用者政策。
- **对照、反转条件及验证建议（NOT_RUN）：** 固定小响应可整缓冲；大/未知长度按流与实际 framing；对照 AB-010 FFmpeg 非一进一出及 drain。验证短读、长度不符、写失败、close 错误、取消与背压；未运行传输。

## 实际来源登记

- **T06-S01**：[aria2/aria2/src/DefaultBtProgressInfoFile.cc](https://github.com/aria2/aria2/blob/release-1.37.0/src/DefaultBtProgressInfoFile.cc) · ref `release-1.37.0` · Blob `7907284feb296ebef39940b28d499266c1e18304` · L111–124,234–300。进度文件版本/字节序、条件 InfoHash 及分片长度检查。
- **T06-S02**：[curl/curl/docs/libcurl/opts/CURLOPT_RESUME_FROM_LARGE.md](https://github.com/curl/curl/blob/curl-8_11_1/docs/libcurl/opts/CURLOPT_RESUME_FROM_LARGE.md) · ref `curl-8_11_1` · Blob `061890dd8a1ee2edb8271e9954fc4f86a653e2fb` · L29–42。官方选项文档，偏移和 FTP 上传追加语义。
- **T06-S03**：[syncthing/syncthing/lib/model/folder.go](https://github.com/syncthing/syncthing/blob/v1.29.0/lib/model/folder.go) · ref `v1.29.0` · Blob `f8b8c501f06eecfcb2aa95c34beec83e9c0f5e75` · L39–54,147–163。folder 资源、配置、文件集合和 Serve 生命周期。
- **T06-S04**：[restic/restic/internal/restic/snapshot.go](https://github.com/restic/restic/blob/v0.17.3/internal/restic/snapshot.go) · ref `v0.17.3` · Blob `39ed806276ab54d304a1b3b72fecf4a9d1123681` · L14–30。Snapshot 时间、parent/tree ID、路径、排除项。
- **T06-S05**：[python/cpython/Lib/csv.py](https://github.com/python/cpython/blob/v3.13.0/Lib/csv.py) · ref `v3.13.0` · Blob `cd202659873811a2ea47c7374cc268d26a9ac5f3` · L145–159,199–214。DictReader/Writer 的字段映射、长短行和额外字段策略。
- **T06-S06**：[golang/go/src/net/http/transfer.go](https://github.com/golang/go/blob/go1.23.4/src/net/http/transfer.go) · ref `go1.23.4` · Blob `5a3c6ceff5780832728f214b4caed10241e4cfb7` · L57–74,336–351。运输帧状态、未知长度及 BodyCloser 清理。

跨分区来源：[T05-S08/S10](T05-common-function-abstractions.md)、[T04-S03/S04](T04-intelligence-and-infrastructure-architecture.md)。T06 DELIVERED_SCOPED；真实 HTTP Range/ETag 合同、文件同步冲突算法、备份完整性、批量事务和查询实际回归仍需后续深查，不声称已验证。