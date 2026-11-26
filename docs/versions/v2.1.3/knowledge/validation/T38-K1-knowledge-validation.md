# T38-K1 · 最小知识正确性与适用性核验

#638 / #563；冻结输入 c2bc7d379dd1b83e23d55da390ba4b8c2f10f3f5；[最新 Owner 方向](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7190227558)。T37-R5 G0 限定通过。当前阶段 **KNOWLEDGE_CONSTRUCTION**，验证对象是知识的正确性、合理性和适用边界，不是 Tally/DomainHarness 的 Package、SDK 或产品实现。原 G1–G4 产品方案保留为历史规划，暂停实施准入；#565 四项发现 OPEN/HOLD、#286 DEFERRED、A24/T4 NOT_RUN/HOLD。本次不用模型A/B、Agent、CI、第三方库，不建复用引擎、测试平台、Package 或新知识Rule。

## 1. 有限方法

按具体知识问题选最小材料：确认原文及来源资格→明确教学合同、正例/反例/合法变体→只有机制可观察时使用最小单文件代码→区分实际观察、演绎及未证推断→记录知识缺陷及归属。优先文字或数学推导，代码并非所有知识的必要形式。工程抽象选择需要真实变化/Owner/消费者/成本事实，不能把某段代码通过就说抽象全局最优；独立审核须从明确合同与原文推导预期，而非以作者断言为唯一Oracle。

## 2. 本批冻结原文与判定

| ID | 原知识（冻结 path/blob/含端点行） | 独立于示例实现的教学合同及预期 | 实际证据与知识裁定 |
|---|---|---|---|
| A | v2.1.2/software-engineering-knowledge/K02/advanced-engineering-abstraction-boundaries-and-variation-r19.md `17c7dabe9ca7c2478dfb182231b3ab96e2eb7cbc` L17–48；K03/advanced-abstraction-level-selection-and-semantic-reuse-r19.md `a1e640efb9866f1981a80f43d64c661635d52c64` L7–38（路径相对 docs/versions/） | 小整数金额[120,80]合计200；预览无签署允许；发布必须有签署且获许可。共同纯计算可以共享而不能合并完整行为；若真实只有一个局部调用且没有变化收益，内联也合法。 | A 实测预览200、未签署/无许可拒绝、合法发布计算200、错误合并返回200，后者在本教学发布合同下反例成立。仅模拟发布入口返回值/拒绝，不执行真实发布或证明完整权限系统。共享计算可保持差异；不证明必须抽helper或某个成本最优；纯计算同变化及共同维护事实在真实任务另证。限定机制支持。 |
| B | 同K02 L49–54（R19-F2反馈候选）；v2.1.3/knowledge/fine-grained/T27a-detailed-design-representation.md `0e1e0bd6db29c7d73626d1a502a08f0ca845e83a` L29–49 | 初始区间对象hi=2；调用者合法改为9；快照合同要求观察仍2，live合同要求9；若未声明所有权，不裁快照或live优劣。 | B 实测浅复制9、元素级复制2、直接引用9；针对这个扁平对象及一次写入，不证明任意深层对象复制、冻结、排他所有权或一般并发安全。明确反例支持输入引用与内部状态Owner不同，不晋升反馈为实证效用。 |
| C | 同T27a L18–28、L29–49 | 数学集合定义为区间并集，RI另要求闭区间有序且不相邻；[(1,3),(7,8)] 与 [(1,2),(3,3),(7,8)] 集合相同，后者违反指定RI。若业务需保留原始区间来源/分段，两表示不能仅凭集合相等合并。 | C 实测集合相等、RI true/false，分段不同；只核两个小整数输入与独立纸面集合{1,2,3,7,8}，不证明归一化算法、极值或跨语言算术安全。源RI支持，合法变体为改变抽象值合同，不是放宽原RI。 |
| D | v2.1.3/knowledge/fine-grained/T27c-1-concurrency-task-lifetime.md `8fb9513b0bca07e4f81bb0750ff905c3ca87c755` L5–7 | 同一JS事件循环一个可用名额，两任务检查后await；若两次检查均见0，之后分别写入会得到2；若在同一同步段重查及增加，最多1。若真实有两个名额，允许两个成功，不将1写成普适限制。 | D 实测无重查[true,true]/used2，有重查[true,false]/used1。只证明指定单线程Promise交错；未执行Go内存模型、多线程、跨进程或数据库。合法容量变化是纸面反转，非代码已执行，实际同步边界另核。源逻辑竞态机制支持。 |
| E（纸面，不运行） | v2.1.2/software-engineering-knowledge/K05/advanced-quality-reviewer-calibration-r18.md `8ae8fe2b652407aa300bf60336c0e7930b42b992` L51–55 | B 若任务合同明确快照，外部写使观察9是有权反例；若明确live，则9符合；若未约定，快照断言属于ENGINEERING_HYPOTHESIS，不能据此主PASS/FAIL。A 的wrongMerge亦须绑定发布而非预览合同。 | 作者按声明事实演绎，非运行独立Reviewer/Validator、隐藏实验或测量发现率。保留实际A–D观察，未声明所有权时结论UNKNOWN，不把同一行为重新标为已证缺陷。待独审核合理性。 |

所有文件完整路径前缀为 docs/versions/；原实际源在冻结输入读取核对。A–D 为最小合成教学见证，不是新增登记真实Case或通用机制定义；原来源、Claim/Case、规则资格不变。E 的 Oracle 资格是纸面判断，NOT_RUN。本文不证明LLM加知识后效果提升，也不测量召回/质量/维护成本。

## 3. 实际可重复执行收据

单文件 [T38-K1-witness.mjs](T38-K1-witness.mjs)，43行左右（以实际文件为准），只有Node内置 assert。实际临时目录执行：

```sh
node --version
node --check witness.mjs
node witness.mjs
git hash-object witness.mjs
sha256sum witness.mjs
```

实际 Node v22.16.0；语法检查与执行 exit 0，实际输出：
```text
A preview=200; unsigned/unauthorized publish denied; authorized=200; wrongMerge=200
B shallow=9; snapshot=2; live=9 (legal for live contract only)
C AF equal; RI canonical=true split=false; original intervals differ
D check-before-await: 2 accepted; recheck+increment without await: 1 accepted
4 fixture groups observed as expected; not product or knowledge-wide PASS
```
Git blob `eda008c46cf42ccca41f907e2c335e6a3e3ee99e`，SHA256 `538d0435a619fc70aca08252505a75e5d1866ec5831d3a40cff4d16d0a1f4339`；执行原文件与 GitHub 已提交实际 Blob 核对相同。仓库内文件名不同，复现直接 `node --check T38-K1-witness.mjs && node T38-K1-witness.mjs`。所有断言为固定有限输入，不是工程测试全覆盖。观察到错误候选的预期错误行为并非错误候选通过业务合同；断言通过只表示见证可区分。

## 4. 反馈与下一批

当前未发现这四个有限机制及E条件推导与原文冲突；不修改专业正文，不将原研究候选晋升、也不宣称全部知识验证。将独审集中在实际源含义、教学合同是否足够、反例/合法变体和独立预期、实际脚本身份与执行证据及未证边界；独审可使用同一文件、手算、或更小独立探针，不要求SDK/Package或环境建设。如果知识错误，修原Owner正文/条件/关联；若仅见证错误，修见证；若未知事实，保留UNKNOWN。后续按真正待核疑点分批（关系/事务及协议、任务状态及失败恢复、业务政策组合、工程验证与演进），不以每条引用都写脚本、全量测试平台或增加引用数量为目标。当前67/86、46/265、S51/D0/U5及历史资格保持。下一阶段仍知识建设。
