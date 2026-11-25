# T29a-1 · 真实缺陷与修复：局部语义、跨组件合同及回归

#592 / QA05/06；K06 主责诊断与修复，K05 回归 Oracle，K03 语言及生命周期机制。复用 [K06 卷一](../../../v2.1.2/software-engineering-knowledge/K06/foundations-deep-dive-01.md) 症状/直接失败/根因、[卷二](../../../v2.1.2/software-engineering-knowledge/K06/foundations-deep-dive-02.md) 竞争假设及因果证据、[T28a-1](T28a-1-test-design-stateful-isolation.md) 独立测试。以下两组是真实历史 issue 与实际提交差异的专业解读，不是新增 T22 catalog Case 或已合格 Claim。未复现、未执行上游测试，NOT_RUN。保留报告者观察、维护者判断、实际源码变化及我们的推断之间的界限。

## 1. 如何读历史缺陷而非仅复述修复摘要

记录准确报告对象、发生环境/输入、现象、应有行为的依据及其争议、引入变更（若只有 bisect 报告注明）、修复 SHA 与实际代码差异、测试差异、未验证条件。回归测试应能区分引入缺陷与修复，最好在固定前后版本执行；本轮只有静态阅读，不能把存在断言当已验证前 FAIL 后 PASS。若修复同时改变旧可观察行为，明确变更范围与相容性。若回退修复引入的回归，不必声称原问题也解决。多仓库交互要核实际调用/生命周期协议，不因一方最后抛错就归咎那一方全部。

## 2. D01 CPython 生成器表达式多次调用迭代协议

### 证据

- [issue #127682](https://github.com/python/cpython/issues/127682) 2024-12-06：报告 CPython 3.13.0 与 3.13.1 的观察差异，Linux 自编译与 Windows 官方下载均报告额外 __iter__ 调用。报告者使用有日志副作用且返回自身的迭代器进行对照；输出值仍可能正确。维护讨论将变化与先前防崩溃修复相关联；评论报告 bisect 到 bcc7227（缩写，未独立重跑/展开该 SHA），为 #125178 的 backport。
- [PR #125178](https://github.com/python/cpython/pull/125178) 合入 079875e39589eb0628b5883f7ffa387e7476ec06，实际读 PR 描述：增加迭代相关检查，涉及生成器局部变量修改、TypeError 及测试。这里只使用维护者记录，不宣称独立核验全部引入 patch 或完整崩溃因果。
- [PR #132351](https://github.com/python/cpython/pull/132351)，实际修复 commit [d87e7f35297d34755026173d84a38eedfbed78de](https://github.com/python/cpython/commit/d87e7f35297d34755026173d84a38eedfbed78de)。[3.13 backport #132384](https://github.com/python/cpython/pull/132384)，实际 commit [132bdf6990003df61d30c379a12c041010d00245](https://github.com/python/cpython/commit/132bdf6990003df61d30c379a12c041010d00245)。两份提交实际读取以下差异：

| 位置 | 实际改动 |
|---|---|
| main Python/codegen.c，3.13 Python/compile.c | 在 comprehension 的外层参数准备中，移除产生迭代器的 helper 调用，改为访问 outermost->iter 表达式，再调用生成器函数 |
| Lib/test/test_dis.py | 两处反汇编期望删除外层 GET_ITER |
| Lib/test/test_generators.py | 新增 test_genexpr_only_calls_dunder_iter_once：外层 C 的 __iter__ 返回仅具有 __next__、没有 __iter__ 的 Iterator，消费结果期望 [1,2] |
| Lib/test/test_genexps.py | 删除非 iterable 整数在构造表达式时立即抛 TypeError 的 doctest |
| NEWS | 明示避免重复 __iter__，非 iterable 错误延迟至执行生成器表达式时 |

### 机制与工程判断

生成器表达式的外层参数先准备与生成器内部实际迭代是两个边界。先前检查修复可使内部再执行迭代获取；如果外层也先取一次迭代器，重复调用具有可观察副作用，或对仅具有 __next__ 的返回对象再次要求 __iter__ 导致异常。修复实际删除外层获取指令、保留 outermost 表达式求值，改变了迭代器建立/非法输入报错的时机；不能称完全没有可观察语义变化。原报告打印两次的现象与修复目标一致，但新增测试没有直接统计原日志调用次数，主要验证缺少 __iter__ 的返回对象可被生成器表达式消费；反汇编与 NEWS 提供互补证据。不能从单个新增测试推断所有异步/嵌套生成器或历史 crash 路径验证成功。

原创补充 Oracle（NOT_RUN）：有副作用计数的 iterator，分别观察构造、首次 next、遍历和终止；仅 __next__ 返回对象的有效消费；非 iterable 在构造与执行两个点的异常合同；保留原防崩溃场景的既有测试，不能为了移除重复调用删除安全检查。选择新的结构性修复而非直接回退原防崩溃修复；该选择是本章工程解释，不替维护者声称其全部意图。与 T25b-1 求值/副作用、T27a 行为合同相关。

## 3. D02 pytest / pytest-xdist 提前停止、fixture 重复生命周期

### 证据链

- [pytest #11721](https://github.com/pytest-dev/pytest/pull/11721) 合入 12b9bd580198edf88a795b692cbd6a1a05fe408a，原目标 #11706：提前结束时使高作用域 fixture teardown 错误得到正确报告。实际读 PR 描述与下述 revert 差异，未重跑原提交。
- [xdist #1024](https://github.com/pytest-dev/pytest-xdist/issues/1024) 2024-02-22 的报告：两个 worker、maxfail=1 时 session fixture setup 被观察三次，不加限制时两次。维护者确认并报告 bisect 至 12b9bd5；讨论也展示 pytest 7.4.4、xdist 3.5.0 的单 worker 仍可能继续执行一个测试。注意不同观察不是同一次实验；报告者描述 setup，pytest #12021 描述 teardown 两次，不能合并成同一计数。
- [pytest #12021](https://github.com/pytest-dev/pytest/issues/12021) / [#12022](https://github.com/pytest-dev/pytest/pull/12022)，合入 [93cd7ba857471dc9f2322246f37ec5eb10cbc91c](https://github.com/pytest-dev/pytest/commit/93cd7ba857471dc9f2322246f37ec5eb10cbc91c)（2024-02-23）。实际 diff 在 src/_pytest/runner.py runtestprotocol 移除 shouldfail/shouldstop 时将 nextitem 置 None 的分支；testing/test_runner.py 删除两个针对 --maxfail=1 和 --stepwise 的原错误报告回归测试；changelog 明示 8.0.2 回退，PR 明确重新打开 #11706 并保留 sticky stop/fail 变化。没有新增 xdist 回归测试的事实不能改写成已有测试通过。
- [xdist #1026](https://github.com/pytest-dev/pytest-xdist/pull/1026)，合入 [84df4451ede720f09b0c7780a3f5f739b2271f83](https://github.com/pytest-dev/pytest-xdist/commit/84df4451ede720f09b0c7780a3f5f739b2271f83)（issue 记录 2024-04-01 关闭）。实际 diff：remote.py worker 在一次 run_one_test 后若 session.shouldfail/shouldstop 则退出循环，sessionfinish 发送两字段；dsession.py workerfinished 读取字段，传递停止原因，否则走正常节点移除分支。acceptance_test.py 新增 test_maxfail_causes_early_termination：单 worker 两个测试，一个失败、一个可通过；运行 --maxfail=1 -n 1，预期 failed=1（assert_outcomes 默认其他计数依实际框架合同；不宣称独立执行）。原双 worker -x 测试改为允许 1–2 个失败，移除原对通过数/总数的部分限制，注释明确并发时序。

### 机制与判断

原 pytest 修复把 nextitem=None 作为当前会话将结束的信号以触发高作用域 teardown。但 xdist 有自己的 worker 调度与全局失败上限协调；若 worker 已根据局部标志清理，却仍进入后续测试，可能再次建立/清理 fixture。该解释与 diff、报告及讨论吻合，未独立重建完整 hook 轨迹。核心不是 fixture 天然不能执行两次，而是生命周期所有者、允许继续执行的条件和分布式停止协议不同步。单 worker 也可能有 controller/worker 交互，故不能只研究线程数。全局 maxfail 是并发协调目标，正在执行的多个 worker 可能同时失败；不能机械断言任何全局执行恰好一次失败。

修复路线分两步：先回退破坏兼容的局部行为（原报告问题重新打开），再在实际 worker/controller 协议修正退出和报告。现有 xdist 新测试直接约束后续测试不继续，不直接计算 fixture setup/teardown 次数，补充验证可在独立 fixture 事件计数下核完整生命周期，并保留原 teardown 错误报告的独立预期；本轮不宣称 #11706 已最终修复或重新引入原 pytest 变更。

教学 Oracle（NOT_RUN）：固定 pytest/xdist 组合，比较无插件、-n1、-n2，记录 worker ID、setup/teardown/实际测试调用、停止原因；对原失败版本验证后续测试是否执行，对回退版核回归缓解及原错误报告限制，对 xdist 修复核停止字段和终态；区分用户 fixture 非幂等与控制层重复进入。并发版本允许合法交错而非要求日志严格全序。复用 T27c-1 task lifecycle、T28a-1 fixture 与独立 Oracle。

## 4. 可迁移的故障类型与反例选择

| 故障类型 | 从实际历史可提取的诊断问题 | 非通用结论 |
|---|---|---|
| 隐藏副作用/重复求值 | 调用次数、时机及返回值相同是否足够 | 不是所有 __iter__ 重复都造成同一缺陷 |
| 修复引入回归 | 原修复的安全/错误报告目的是否仍需保留 | 不可一律 revert 所有复杂修复 |
| 跨组件生命周期 | 谁宣布结束、谁仍能调度、谁清理资源 | 不可把局部 stop 当整个分布式停止 |
| Oracle 不充分 | 最终值正确是否掩盖调用、身份、资源或 Effect | 不可用一个输出快照替代所有合同 |

真实历史记录作为专业学习材料，不晋升旧 Case 目录或 Source Claim；阅读两组修复 diff、回归测试差异与真实 issue/PR，未运行旧版本或修复版，未核全部相关源码、完整 release 包或 CI。后续 T29a-2 应扩展到数据/文件/协议等不同故障族、实际最小反例及更广证据；不以仓库数量替代深度。
