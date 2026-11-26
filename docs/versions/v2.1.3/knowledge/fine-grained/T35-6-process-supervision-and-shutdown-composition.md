# T35-6 · 跨进程资源所有权、监督与异常退出（G5）

#646 / T39-R2 候选增量 / RE-02；K03 运行机制、K02 所有权及边界、K10 运行恢复、K05 独立评价。基线 8b4594b1a27852c294a379e345bf6f59451c7545；KNOWLEDGE_CONSTRUCTION。复用 [T25c-1](T25c-1-os-process-memory-ipc.md) Blob 886d44bec7756479dde4363820265907bd640297 L11–37 的 spawn/exec/wait、FD与pipe；[T27c-1](T27c-1-concurrency-task-lifetime.md) Blob 8fb9513b0bca07e4f81bb0750ff905c3ca87c755 L34–40 的停止准入和协作收敛；[T35-3](T35-3-complex-general-function-composition.md) Blob 6d3d81acd9fa952ebc3f06e1af9d5e4250822f89 L75 明列跨进程关闭未选。仅补进程监督的具体身份与集合闭合、监督者故障和输出/效果边界；不重写普通 IPC 或创建通用 supervisor 产品。

## 1. 先定义所拥有的工作，而非根据进程名称追杀

教学合同：一次获权转换任务 W 启动一个主程序，允许其启动有限子程序；所有受控本地执行者必须停止才释放 W 的执行名额，输出文件另核完整性，外部效果另核权威。进程对象、任务ID、启动尝试、进程集合和实际授权分别记录。PID、命令行或祖先链的快照均不是可跨时间复用的授权身份；退出后 PID 可再分配。仅管理一个不创建后代的程序时，持有真实进程对象、读取双流和回收已经足够，不默认建进程树机制。

Linux pidfd_open(2) 可取得特定进程的文件描述符；pidfd_send_signal(2) 对该引用发信号，可避免持有有效旧引用时因 PID 重用误伤新进程；获取引用之前的 PID 来源仍须可靠，不用迟到的数值 PID 猜旧身份。pidfd 可由 poll 等观察退出；waitid 对调用者的子进程才可按对应等待合同取得状态，不能将所有外部进程变为自己的子进程。单进程引用不表示全部后代，发送信号成功不表示目标已终止。

## 2. 集合成员资格与监督者失效

Linux 6.7 cgroup v2 文档：非根 cgroup 的 cgroup.events 中 populated 反映该子树是否仍有活进程；cgroup.kill 写1用于终止适用非线程型子树的进程。应先由获权控制者建立任务专有的实际 cgroup、控制加入/迁移和创建路径，并停止新准入，再对该受控集合执行所需动作及观察 populated=0。不能把进程组或父子列表等价为此集合，也不能把 cgroup 机制称为完整安全沙箱。若任务成员可以自行逃出受控子树、监督者错误分配或后来仍允许新进程进入，空集合观察不足以证明整个任务永不再运行。populated=0 不负责应用子进程回收、远端效果和文件提交；需分别核验。具体 kernel 版本、委派权限、服务管理器清理合同未获证为 UNKNOWN；监督者死亡不会仅因存在 cgroup 就自动触发 cgroup.kill。

Windows Job Objects：获权进程可以关联至 job，实际后代继承及 breakaway 由创建方式、job限制和嵌套关系决定。Windows 8/Server 2012 起支持嵌套 job。启用 KILL_ON_JOB_CLOSE 时，最后一个 job 句柄关闭会触发成员终止；仍有被复制/继承的句柄则“监督者退出”不等于最后句柄关闭。不能从普通 job 对象 wait 信号泛推任意终止完成；Microsoft 文档对 job signaled 的描述是特定 end-of-job time limit 事件。选用真实 API 查询成员与进程终态，阻止新准入及不允许的 breakaway；记录失败、未证对象和清理范围。进程启动后才分配 job 可能在两动作之间已运行并创建不受管成员，应核实际创建期关联或受控启动方案及失败窗口，不宣称任意 API 原子；未执行 Windows 实验。

## 3. 停止与完成是多维观察

教学停止序列：关闭新增 W 准入，取消或保留待办，发协作停止请求，等待在途工作至有权期限；剩余工作按明确权限强制终止受控集合，继续有界观察退出、回收、stdout/stderr终态及资源清理。强制操作不执行任意语言 finally 的保证，持久文件可能部分写入，必须按原 staging/receipt 合同恢复。deadline 只界定等待与升级，不能承诺操作系统在固定毫秒内收敛。监督者崩溃后由独立实际外层机制决定成员终止或继续；重启监督者不得从本地旧PID列表认领新进程。持久 task/attempt 与资源权威绑定，已失身份时保留 UNKNOWN 并停止冲突重派，依据独立权威确认或获权接管。文件/DB 写许可在实际资源边界撤销或 fencing，不能用终止信号替代。没有跨进程控制需求的纯函数或协作组无需上述全部结构。

| 教学历史（NOT_RUN） | 独立判断 |
|---|---|
| W 主进程退出0、孙进程仍持 stdout 写端 | 主进程已退出；流未读尽且集合未收敛，不报告 W 完成 |
| 旧 PID 500 退出后新任务获得500 | 不依据旧数值终止新任务；真实旧引用不可被新任务偷换 |
| 监督者死、另有 job 句柄持有者 | 不推出 kill-on-close 已触发；核实际最后句柄与成员状态 |
| cgroup 为空但仍允许后续迁入 | 只证明该观察时刻活成员为空，不证明永久闭合 |
| 成功强制终止但外部写已发生或未知 | 进程已停止与业务效果独立；查权威，不自动补偿/重发 |
| 输出超过上限且读取方关闭 | 按合同处理 broken pipe/任务失败，保留原错误和输出截断，不能凭截断日志判完整 |

## 4. 来源与边界

2026-11-26 实际核对 Linux [pidfd_open(2)](https://man7.org/linux/man-pages/man2/pidfd_open.2.html) L110–137、[pidfd_send_signal(2)](https://man7.org/linux/man-pages/man2/pidfd_send_signal.2.html) L123–132；Linux [6.7 cgroup v2](https://docs.kernel.org/6.7/admin-guide/cgroup-v2.html) L129–133、L443–448；Microsoft [Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects) L81–95。定向读证，不宣称全 API/平台一致或目标执行。所有组合/Oracle NOT_RUN；不改变原知识资格，不实现产品、Package、SDK或安全专项。
