# T25b-2 · 时间语义：时钟、期限、日历与业务生效

#594 / FD04；K03 计时/表示，K01 业务有效期，K02 状态合同，K06 因果诊断、K10 调度与恢复。复用 [T25b-1](T25b-1-language-value-and-text-semantics.md) 数值单位、[K01 STK](../../../v2.1.2/software-engineering-knowledge/K01/advanced-stakeholder-authority-and-triangulated-evidence.md) 来源与生效范围、[K06 基础卷二](../../../v2.1.2/software-engineering-knowledge/K06/foundations-deep-dive-02.md) 因果顺序。以 Linux 时钟与 Python3.11 文档为具体实例，所有例及 Oracle NOT_RUN。

## 1. 不同时间对象

| 对象 | 问题 | 例 |
|---|---|---|
| Instant/时间点 | 事件在所选时间尺度中的何处 | UTC 时间点 |
| Duration/经过时间 | 从一个测量点到另一个经过多少 | 请求超时预算 |
| Civil/local date-time | 某地日历与时刻 | 用户约在当地09:00 |
| Calendar period | 按日历规则跨日/月/年 | 下个月账单日 |
| Effective/valid time | 业务事实何时适用 | 合同7月1日起生效 |
| Recorded/transaction time | 系统何时获知或记录该事实 | 7月3日收到追溯更正 |

ISO 格式只是表示，不自动解决时区、时间尺度、精度或权威事实。一个偏移如 +06:30 说明给定表示与 UTC 的差，不描述地理时区的未来/历史转换规则。只存 UTC 适合很多已发生事件，但对“当地每周一9点”还需保留时区标识及日历规则；否则无法重建用户意图。只有日期的生日/结算日也不一定是一个 UTC 瞬间。墙钟时间点、经过时间和业务生效不可互换。

## 2. 时钟选择、精度与期限

Linux CLOCK_REALTIME 代表可调墙钟；手动校时等可能产生不连续变化。CLOCK_MONOTONIC 不随墙钟突跳倒退，但可受频率调整影响，Linux 不计系统 suspend，连续调用可能返回相同值。CLOCK_BOOTTIME 适用于需要包括挂起时间的某些 Linux 经过时间合同。进程/线程 CPU time 衡量 CPU 执行，不是请求墙钟延迟。选择取决于任务：显示/跨系统事件记录用时间点；同一时钟域测量经过时间使用适当 monotonic；需要睡眠期间也过期必须核目标 API 对 suspend 的行为。

分辨率是最小可区分间隔，精度/准确度、同步误差与调度实际唤醒延迟是不同概念。纳秒表示不证明纳秒准确；定时器到期不代表 callback 立即执行。时间差使用同一单位、同一时钟域，防溢出/舍入；monotonic 绝对数不能作为跨机器、跨重启持久时间点。

教学 deadline（同一 clock 域，固定有限预算）：
```text
deadline = now_monotonic() + budget
last = NO_ATTEMPT
for attempt in finite_attempts:
    remaining = deadline - now_monotonic()
    if remaining <= 0:
        return waiting_timeout(last)  // preserve known/unknown Effect
    last = perform_next_step(timeout <= remaining)
    if completed(last):
        return observed_result(last)  // late completion not on-time
    if not retry_allowed(last) or no_attempt_left:
        return terminal_attempt_result(last)
    remaining = deadline - now_monotonic()
    if remaining <= 0:
        return waiting_timeout(last)
    delay = min(authorized_backoff, remaining)
    if delay > 0:
        wait(delay)
    // next iteration recomputes remaining before any work
return terminal_attempt_result(last)
```
每次重试不得重新取得完整 budget；每个实际调用与 backoff 前重新计算剩余量，等待后再核验。超时结果保留最后已观察状态，Effect 无权威证据为 UNKNOWN；不得返回确定未执行。已收到真实迟到结果需保留，但不能报按期完成。无重试或次数用尽时明确最后观察与未收敛状态。伪代码不是能强制停止所有阻塞调用的实现；目标 API 必须真正支持 timeout/取消，达到期限只说明等待边界，实际任务、资源、远端结果另按 T27c-1、T25c-2 确认。例：在5秒预算中，第1阶段消耗3秒，第2阶段最多剩2秒（未计其他开销）；若第2阶段再耗2秒仍未完成，剩余0，不得启动第3阶段或正时长 backoff（NOT_RUN 反例）。若墙钟此时向后调一分钟，使用 realtime 差会误延长等待。若设备 suspend，MONOTONIC 是否满足“5秒后必过期”取决于业务是清醒时间还是实际经过时间。

## 3. 时区、夏令转换与日历算术

Python3.11 zoneinfo 基于 IANA 时区数据，默认读取系统数据或 tzdata 包。实际版本与数据来源要记录，用户在不同环境看到的未来计划可能不同。naive datetime 不带明确时区语义；仅附加 tzinfo 并不自动验证当地时间真实存在，也不同于从明确瞬间转换时区。PEP495 把时钟回拨形成的重复区间称 fold，前跳缺失区间称 gap。教学取官方例 America/Los_Angeles 2020-11-01 01:00，fold=0 对应 -07:00（08:00 UTC），fold=1 对应 -08:00（09:00 UTC）；两者本地文字相同但瞬间不同。对前跳 gap 的当地时间，不能用 fold 自动创造实际存在的时刻。需要明确业务策略：拒绝、询问选择、移动到合法时间等；不能自行设定。

校验教学：对输入的 local date-time 与已确定 zone，构造候选 offsets/fold，分别转 UTC 后再转回 zone，检查本地字段与候选瞬间；以转换后的 UTC 瞬间键去重（不能用同 tzinfo 的本地 datetime 相等去重），0候选为 gap、1为唯一、2为歧义（适用所述 Python zoneinfo/PEP495 两候选模型，不推广任意时间库）。例如 LA 2020-03-08 02:30，两个 fold 候选转 UTC 再转回分别为03:30和01:30，均不匹配，零候选。保存选择、zone、offset 及必要时规则数据版本。Python3.11 的 datetime 相等比较在两个 aware 对象共享同一 tzinfo 时忽略 fold；减法也忽略该共同 tzinfo 而按本地字段计算。上述两个01:00 若共享同一 ZoneInfo 对象，直接比较可相等、减法可为0；转成 UTC 后比较/相减才反映这两个不同瞬间的一小时差值（教学推导 NOT_RUN）。需要实际经过时间时先转换成适当统一瞬间或使用 monotonic 观测，不能以相同本地字符串为身份。历史事件的解释与未来用户计划的再计算策略分开。

日历算术：每个日历日不总是24小时；每月不等于固定30天。若规则为每月最后一天，应按年月历法求最后有效日；若为每隔720小时，应按 duration，二者可不同。教学约定：1月31日加一个月，业务可能要求2月末、拒绝、向后溢出或保留原计划日供未来月份计算；没有授权不得自动选择。跨 DST 每天当地09:00 与固定24小时间隔也不同。例如 Python3.11 同一 LA ZoneInfo 下 2020-03-07 09:00 加 timedelta(days=1) 得3月8日09:00，UTC 分别17:00和16:00，相差23小时（按该历史规则，纸面 NOT_RUN）。日期范围查询优先按实际合同使用半开区间 [start,end)：相邻区间共享边界但不重复计入；当地自然日先按 zone 计算两端瞬间，不能假设 UTC start+86400秒始终是当地下一日。闰秒、POSIX epoch 与 UTC/TAI 等时间尺度的严格转换留具体领域，不声称所有库能表达闰秒。

## 4. 业务有效期、观察历史与事件顺序

示例：价格规则有效期自 7月1日开始，系统 7月3日才记录纠正。7月2日按当时已知事实处理的订单与7月4日回看7月2日的纠正事实，可能产生不同查询结果。若系统需要两类审计，应分别保留 valid time 与 recorded time/version，不用一次 UPDATE 抹去历史。区间起止、是否允许重叠、未来生效、撤销、更正追溯权、时区均由业务责任人确定。历史业务状态不能凭新配置自动重写。

多主机 wall clock 可能偏移，观察时间和真实发送/接收事件不是同一事实。K06 卷二已有 happens-before 及 trace 边界；这里补时间字段设计：记录事件发生时间及其来源、接收时间、逻辑顺序/版本（若有）、关联身份、可能的不确定性。消息序号只在定义范围内有序；不同 producer 的局部序号不可直接比较。若需要全局排序/共识，必须另有实际协议，不把时间戳最大等同最新权威值。租约还依赖权威时钟及旧持有者 fencing，不能仅用客户端墙钟判断当前权限；完整分布式机制留后续。

## 5. 调度、重复执行与恢复

日历计划规定何时应触发；实际任务可能因停机、时钟变化、挂起、负荷而迟到。业务需明确错过时跳过、合并、补跑所有或有界补跑，重复时如何识别计划 occurrence（例如规则身份+已解析瞬间/合法本地 occurrence 区分 fold）。同一计划的一次触发不是同一任务已成功，状态、重试、结果及实际发生需分别记录。计划变化应保留版本与生效范围；不将旧任务自动当新计划。跨重启 TTL 需持久化可解释的绝对期限及其时钟信任，重新启动按真实合同重建剩余时间；不得保存原 monotonic 数字并在新 boot 直接比较。

## 6. Oracle 与来源

| 教学反例（NOT_RUN） | 独立预期 |
|---|---|
| 墙钟向前/后跳、系统 suspend | 期限使用正确时钟域；不误报资源收敛 |
| 两次 retry 与跨阶段，5s预算3s+2s未完成 | 不再调用下一步或正时长 backoff；区分等待已停止与任务未收敛 |
| fold 两个01:00、gap 不存在时刻 | 区分两个真实瞬间与零候选；选择策略须获权 |
| 1月31日、闰年2月、自然日跨 DST | 按明确日历合同，不能默默固定30天/24小时 |
| 追溯更正与未来生效 | 同一有效时点、不同观察版本的独立期望 |
| 停机错过多个触发、重启 | 按获权补跑/去重合同，旧 monotonic 无效 |

2026-11-26 实际读 Linux [clock_gettime(2)](https://man7.org/linux/man-pages/man2/clock_gettime.2.html) DESCRIPTION（CLOCK_MONOTONIC 等时钟特征），Python [3.11 zoneinfo](https://docs.python.org/3.11/library/zoneinfo.html) Using ZoneInfo/fold（本次返回3.11.18在线文档）、[3.11 datetime](https://docs.python.org/3.11/library/datetime.html) 日期时间类型定位、[PEP495](https://peps.python.org/pep-0495/) Terminology/Mind the Gap/Temporal Arithmetic and Comparison。日历策略、有效期/记录时间双维例、期限/调度与 Oracle 为原创工程综合，不声称完整数据库 temporal 标准或全语言行为。具体时区数据库版本、闰秒、跨主机时钟同步与调度实现未实测，DOCUMENT_CANDIDATE，不新增合格 Claim。
