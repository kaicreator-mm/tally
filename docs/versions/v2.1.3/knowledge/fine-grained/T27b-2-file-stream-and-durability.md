# T27b-2 · 文件、流、部分 IO、可见性及持久性

#588 / DA06、RE02；K03.F4.2 主责，K02 数据所有权、K09 迁移、K10 恢复、K05 Oracle 关联。复用 [T27a](T27a-detailed-design-representation.md) 资源/frame、[T25b-1](T25b-1-language-value-and-text-semantics.md) 字节/文本、[T05-C04/C07](../casebook/waves/T05-common-function-abstractions.md) 单值解码及结果迭代。限定 Linux 系统调用与 Node 22.17.0 API 实例；Windows、网络文件系统和不同存储设备须另核实际合同。教学流程与 Oracle NOT_RUN，不是现成可执行的生产文件库。

## 1. 名称、打开对象和数据

路径是命名空间中的查找方式；已打开文件的句柄访问特定打开对象，并不持续重新解析原路径。另一进程可更改文件名、替换目标、改变权限；打开前检查存在与稍后打开不是一个原子操作。Linux rename 的已开文件描述符不因路径更名而失效。读者持有旧句柄可能继续读旧内容；原子路径替换与“每个读者立即刷新”是不同保证。

对配置/导出/上传等功能先定义目标：文件内容、权限/Owner、是否覆盖、可否跟随链接、谁可写、并发版本与目录范围。路径规范化并不自动证明授权或无竞争；在受控目录用实际 API 的排他创建、目录句柄/路径解析合同，并验证打开对象，不能先 access() 再假定 open() 必然安全。Node 22.17.0 fsPromises.access 文档明确提示该竞争。外部路径及符号链接策略须按实际环境决定；本章仅保留工程安全底线，不展开安全专项。

## 2. 一次 read/write 不等于整个任务完成

Linux 普通文件或具有相应 EOF 合同的字节流，read(2) 成功返回实际字节数，正长度请求返回 0 可表示 EOF；特殊设备/消息边界须核实际合同。短读本身不错误。非阻塞资源可能返回暂时不可读错误；信号中断、设备错误等需按 API 处理。write(2) 可能只写一部分；返回成功数不是请求长度，也不等于持久落盘。Node 22.17.0 fs.write 的回执含 bytesWritten，文档警告不要无等待地并发调用同一文件写入。

教学写循环（blocking、独占顺序句柄、有限缓冲、无 append 定位需求）：
```text
offset = 0
while offset < len(buffer):
    n = write_remaining(buffer, offset)
    if n reports transient failure:
        handle only according to actual API + budget
    else if error or n <= 0:
        fail with actual progress; do not report complete
    else:
        offset += n
```
offset 是 buffer 已处理量，不一定是底层文件定位参数；有位置写入时须另计算目标文件偏移。若写了 3/8 字节，再写剩余 5 字节，不能从头重送 8（可能重复）；后续 ENOSPC 时保留已确认进度；某些 IO 错误（例如 direct IO）仍可能修改目标区域，不能把未确认字节判为未修改。对 readExactly(N) 循环累计；提前 EOF 是截断错误，若是任意长度流 EOF 可为合法结束。每次读取使用实际返回长度，不把缓冲尾部残留数据交给解析器。零长度请求不能判 EOF。超时/取消仍可能已产生部分效果；停止后按实际句柄/流完成合同释放资源并处理错误。

## 3. 流、分帧、解析与背压

字节块不等于业务记录。UTF-8 的 U+00E9 字节 C3 A9 若跨两次读取，逐块独立解码可误产生替代字符；应在流结束前维护增量解码状态。JSON 单值解析、完整文档解析、连续 JSON 值、换行分帧是不同合同。CSV 字段可含经引号转义的换行，简单 split newline 不等于 CSV 解析（格式需实际契约）。先确定格式、编码、分帧、EOF、未知字段、输入/单记录/总数据限额以及错误是否允许部分处理。T05 已解释读取一个值不证明整个流合法。

三层进度：传输字节、合法解析记录、业务已提交记录。上传 100% 只能描述计量范围；不能自动证明已验证、已入库或已持久。生产者速率持续高于消费者时，有限缓冲需有等待、拒绝、丢弃或持久排队的真实合同；不能只把输入分块却将所有记录积攒在数组里。小有界配置可一次读并完整验证；大文件选增量处理、控制内存、限制并行与输出顺序。流的 flush/end/close 也不能直接代替文件 sync，具体由 API 和介质决定。

### 3.1 Node 22.17.0 背压协议及终止边界（FG-04）

以 Node.js 22.17.0 的标准 Readable/Writable、单生产者和独占本地临时目标为例。write(chunk) 正常接纳当前块却返回 false 时，它不是拒绝或未写：生产者停止发送后续块，不能重送当前块。等待 drain 仅是恢复发送的条件；错误、提前 close、取消、期限都是不同终态，不能只等 drain。写入返回值还不能证明该块已成功写到底层或持久化。highWaterMark 是反馈阈值，object mode 按对象数、通常字节流按字节数；单块可能大于阈值，Transform/Duplex 两侧各有缓冲，外部任务、解析器、驱动、生产者也可能缓存，因此不是总内存硬上限。限制输入总量、块大小、未完成任务及解析器状态，不能依靠一个阈值解决内存约束。

| 事件/状态 | 观察意义 | 不能推出 |
|---|---|---|
| write 正常返回 false | 当前块接纳，后续生产应等待 | 当前块被拒、应重发 |
| drain | 可按流合同恢复后续写 | 所有业务记录已提交 |
| readable end | 已消费至读取结束 | 所有数据语法、预期长度均正确 |
| writable finish | end 调用后写端数据已交给其底层处理合同 | 文件 fsync、目录持久或远端业务提交 |
| close | 对应流/资源关闭事件 | 一定正常完成 |
| error / 提前 close | 失败，按错误与实际进度处理 | 数据必然没有修改 |

对于标准一次性流，优先使用 pipeline 连接生产、变换及消费，错误和关闭传播由实际 API 管理。不要每次 write false 后另建不受终态管理的监听器。以下是严格 UTF-8 字节流到文本变换的教学路径（目标为已获权的临时文件，额外输入大小限制、目录准备、权限和后续 sync/rename 由原 §4 负责）：
```js
import { pipeline } from 'node:stream/promises';
import { TextDecoder } from 'node:util';

// source: byte chunks; tmpSink: owned writable accepting UTF-8 text
await pipeline(source, async function* (chunks, { signal }) {
  const decoder = new TextDecoder('utf-8', { fatal: true });
  for await (const chunk of chunks) {
    signal.throwIfAborted();
    const text = decoder.decode(chunk, { stream: true });
    if (text !== '') yield text;
  }
  signal.throwIfAborted();
  const tail = decoder.decode();
  if (tail !== '') yield tail;
}, tmpSink, { signal: controller.signal });
// Separate authorized sync/publication/verification step; no publish on rejection.
```
前提 Node build 支持 fatal 选项；若不支持则不可宣称严格拒绝解码错误，先验证能力。若内容有 BOM/字节保真要求，须另定义 BOM 处理及是否允许文本转码；此例不是原始字节复制。块跨编码边界由 decoder 保留状态，最后 decode 检查残缺尾部；仅在整个 pipeline 成功后才进入发布候选步骤。pipeline 的 signal 取消销毁相关流，但自定义生成器或外部工作必须处理 signal，否则可能无法结束；不能认为 abort 调用返回等于全部资源关闭。仅取消 finished 的等待并不会自动销毁流；资源 Owner 需发出适当取消并等待实际终态，处理提前 close 的 ERR_STREAM_PREMATURE_CLOSE 和清理异常。finished 监听器清理策略按实际文档；不要复用状态未知或已失败的流。期限耗尽而清理无法确认时标 UNKNOWN / 未收敛并阻止非法发布，不假装已停止。

| 候选 Oracle（NOT_RUN） | 预期 |
|---|---|
| 慢 sink 使 write false | 每块只接纳一次；生产暂停/恢复；总内存预算需独立测而非 highWaterMark 宣称 |
| false 后 error 或未 end 的提前 close | 失败收据，不无限等 drain，不进入替换目标步骤 |
| 消费者断开/abort、生成器等待外部工作 | 协作取消、任务与句柄终态可观察；未确认时保留未知 |
| C3/A9 跨块；C3 后 EOF | 前者产出 é，后者严格解码失败；不发布旧目标的替换 |
| finish 后文件 sync 失败 | 流完成事实保留，但不宣称持久或业务成功 |

实际资料定位（2026-11-25）：Node22.17.0 [stream](https://nodejs.org/download/release/v22.17.0/docs/api/stream.html) Buffering、writable.write/drain、pipeline async generator/abort、finished/early close、end；[util](https://nodejs.org/download/release/v22.17.0/docs/api/util.html) TextDecoder constructor/decode fatal/stream。Linux read 普通文件 EOF 与特殊设备合同区分，write 已确认进度不保证其余字节未修改；原 #588 独审明确提醒跨挂载点 EXDEV。教学未执行，不新增合格 Claim。

## 4. 原子替换与断电持久性

同一文件系统内将已完整准备的新文件 rename 到目标路径，可避免逐步截断和写入原路径造成读者观察半份配置；前提是实际 rename 语义、正常并发读路径及无其它不受控写入。Linux rename(2) 对已有目标的替换具有命名空间原子语义；已打开旧句柄不改变，跨挂载文件系统可能 EXDEV。文件自身 sync 不必然使包含目录的名称变更持久，Linux fsync(2) 描述需要目录 fd 的显式 sync。存储层实际履行 flush 和目标文件系统支持仍是条件，不应承诺所有故障模型的绝对耐久。

教学方案（Linux、本地支持对应语义的文件系统、同一受控目录、单合法 writer 或已明确并发控制）：
1. 在目标目录排他创建唯一临时文件，设置适当权限；原目标元数据是否保留需显式合同。
2. 写完完整内容，核内容/版本及所有返回值；必要时用独立解析检查实际临时字节。失败时不替换目标。
3. 若要求断电持久，sync 临时文件并处理错误，关闭句柄（关闭失败亦核实际合同）。
4. rename 临时路径至目标路径；不接受跨文件系统 copy+delete 冒充原子替换。
5. 若要求 Linux 该命名变更的持久承诺，sync 目标目录并处理错误。对于非同目录或其他平台，不能直接套用此步骤。
6. 根据阶段与失败实际报告结果；只清理确知属于自己且未发布的临时文件，不因后置步骤失败盲删目标或回滚其他 writer 的版本。

| 故障窗口 | 已知/未知及处理 |
|---|---|
| 写入/临时 sync 失败，尚未 rename | 按上述单 writer 模型旧目标仍在；临时可能部分写入，安全清理/保留诊断 |
| rename 前进程崩溃 | 旧目标不应由此次操作改写，可能残留临时文件 |
| rename 成功、目录 sync 前失败/掉电 | 新名称当前可见，但重启后耐久性未获确认；不能报确定未发生 |
| 目录 sync 成功 | 仅在目标 FS/设备履行所选合同及故障模型内，才有相应持久承诺 |
| 两 writer 同时替换 | 两次都成功也可能最后一次覆盖前一版本；原子替换不是 compare-and-swap |
| rename 返回异常或远程 FS 回执不确定 | 依真实 FS/错误合同判断，可能需要身份/版本读回；不能仅凭异常假设未提交 |

Linux 本地语义不代表 Windows ReplaceFile/MoveFile、SMB/NFS 或对象存储合同。不能仅用 Node API 同名推断所有平台原子性、共享打开、目录 sync 或掉电行为；实际产品需选择可用平台机制并独立验证。

## 5. 竞争、锁和版本

互斥锁的对象、作用域和持有者决定保证：进程内锁不能阻止另一进程，文件锁是否建议性/强制及具体跨进程语义依平台与 API，替换目标后锁住旧 inode 不必然保护新对象。不要用路径文件存在充当永久有效的锁；崩溃、过期回收与旧 writer 恢复有竞争。仅需单合法 writer 时明确其外部保证；多 writer 必须定义如何检测版本冲突、串行化更新或用数据库等权威事务。若用“读取旧版本→比较→rename”且没有共同互斥/原子版本条件，中间仍可被其他 writer 改写。读写者共享合同需明确锁顺序、错误、退出和恢复，不增加无必要锁框架。一般并发机制后续 T27c 深化。

## 6. 导入导出：提交范围及恢复

导入是一条领域任务而不只是文件 parse。可选：
- 小有界数据：完整读取/验证后在适当事务内提交；任何失败保留旧业务状态。不能把远端请求包装在本地事务后冒充全局原子。
- 大数据：先 staging（受控中间数据），分批校验，再按批准的发布合同切换；总量/空间、版本、垃圾清理及失败恢复必须明确。
- 逐项提交：允许部分成功时记录稳定行身份、原始序号/错误与真实提交点，重跑策略按幂等/业务规则；不能以最终 100% 字节数证明所有记录入库。

导出需定义一致快照还是实时游标、分页顺序、过滤/权限、编码/格式版本、是否允许部分文件可见及来源时间。候选断点续传需固定源版本、偏移单位、校验算法/范围和目标身份；源变更不能用旧 offset 拼接。校验和能检出一定范围的意外变化，不证明来源授权、业务合法或持久化。保留合法错误报告，不泄露敏感记录。K09 数据迁移知识及后续 T32b 仍需处理不可逆映射与旧消费者；本章不定义行业政策。

## 7. 独立验证矩阵（NOT_RUN）

| 场景 | 独立预期 |
|---|---|
| 8 字节写回 3，再回 5；或第二次 ENOSPC | 内容与输入独立比较；失败不得报完整，不替换旧目标 |
| 读取 2/4 后 EOF；跨块 UTF-8/带引号换行 | 按固定 framing/编码合同拒截断或正常结束，记录实际合法记录数 |
| 读者持有旧 fd 与替换后新打开 | 在适用 Linux 模型内分别核旧内容与完整新内容；不假定所有人刷新 |
| 文件 sync 与 rename/目录 sync 各处故障 | 区分进程中断与断电模型，核重启后字节/名称、未知阶段及清理 |
| 双 writer、跨 FS、锁持有者退出 | 冲突不得静默覆盖（若合同禁止），不把跨 FS copy 标成原子 |
| 导入部分失败/重复记录；导出期间源变更 | 按选定业务提交范围、独立期望及实际读回检查恢复 |
| 路径更名/权限变动、不同身份 | 不以先前 access 判断当前打开结果，不暴露越权数据 |

正常路径执行成功不等于故障条件已验证；断电/平台测试未执行，须在合法隔离环境按实际目标做。无需为所有临时输出都支付 sync 成本；依据用户承诺、后果和恢复代价选择。

## 8. 来源与限制
实读 2026-11-25 Linux man-pages [read(2)](https://man7.org/linux/man-pages/man2/read.2.html) return value、[write(2)](https://man7.org/linux/man-pages/man2/write.2.html) partial write、[rename(2)](https://man7.org/linux/man-pages/man2/rename.2.html) description/EXDEV、[fsync(2)](https://man7.org/linux/man-pages/man2/fsync.2.html) description/errors；为 Linux API 文档，不是全部 POSIX/Windows/远端 FS 规范。[Node.js 22.17.0 fs](https://nodejs.org/download/release/v22.17.0/docs/api/fs.html) access race/fs.write bytesWritten/同 fd 写入顺序。原案例为历史证据；本文伪代码、故障表、导入导出设计是原创工程综合 DOCUMENT_CANDIDATE，不新增合格 Claim，未运行。Windows/网络 FS、文件锁具体 API/CSV 规范、真实设备断电效果留专项源核查，不能以此文替代。
