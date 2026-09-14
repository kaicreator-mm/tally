# 碰撞率实验(T04)

> T04 的目标不只是「把日志变成指纹」,更是跑那个一直没跑的碰撞率实验:
> 失败日志经流水线后,**同一个错误能不能归到一起,不同的错误会不会被误并**。
> 本文记录第一次真实跑出来的数字。先只做统计,不做修复(修复是 T06 的事)。

## 怎么复现

```bash
go run ./cmd/tally scan internal/logpipe/testdata
```

`scan` 遍历目录下每个 `.log`,按路径第一段选工具链抽取器,跑完整流水线
(收集 → 粗脱敏 → 抽取 → 细脱敏 → 规范化 → 指纹),累加统计。
**同一输入连跑两次输出 byte 级相同**(INV-8,已验证)。

## 数据集

`internal/logpipe/testdata/` 下 16 份脱敏后的真实形状失败日志,覆盖 5 个工具链:

| 工具链 | 样本 | 说明 |
|---|---|---|
| npm | `eresolve.log` `lockfile-mismatch.log` `native-module-abi/`(3 份) | ABI 那 3 份是**同一错误**在不同时间/pid/路径下的记录 |
| go | `compile-error.log` `vet-fail.log` `repeated-undefined/`(3 份) | repeated 那 3 份是同一编译错误跨 3 次 CI 运行 |
| docker | `build-fail.log` `pull-fail.log` | |
| compose | `healthcheck-timeout.log` `container-exit.log` | 并行服务分组 + Python 堆栈 |
| kamal | `deploy-fail.log` `health-wait-timeout.log` | |

数据集是**刻意以「不同根因」为主**构造的 —— 因为比「撞得上」更难、更危险的是
「洗过头导致全撞成一个」(静默故障)。12 个不同根因 + 2 组各 3 份的重复,共 16 份。

## 结果

```
扫描结果
  事件总数:     16
  去重后指纹数: 12
  碰撞率:       25.0%
各工具链分布:
  compose  事件   2  去重   2  碰撞率   0.0%
  docker   事件   2  去重   2  碰撞率   0.0%
  go       事件   5  去重   3  碰撞率  40.0%
  kamal    事件   2  去重   2  碰撞率   0.0%
  npm      事件   5  去重   3  碰撞率  40.0%
```

碰撞率定义:`(事件总数 - 去重后指纹数) / 事件总数` —— 有多大比例的失败事件与先前
事件撞了同一个精确指纹。

## 解读

**两个真正重要的子指标,而不是 25% 这个表面数字:**

1. **真重复归一率 = 100%。** 两组「同一错误多份」(`native-module-abi` 3 份、
   `repeated-undefined` 3 份)各自都精确撞成 **1 个** `fp_exact`,一份不多。
   时间戳、pid、构建路径(/home/alice、/home/bob、/tmp/build-xk29d)被洗掉,
   退出码、包名版本、错误类型、行号被保留。
   - `native-module-abi` 的黄金精确指纹:`27371d8684c00018`
     (见 `testdata/npm/native-module-abi/expected-fp.txt`,测试钉住)。

2. **误并率(假碰撞)= 0%。** 12 个不同根因算出 **12 个互不相同**的精确指纹,
   没有一个被洗过头并到一起。连 `compile-error.log` 与 `repeated-undefined/`
   这种「同一行错误、但整体上下文不同」的都正确区分开了。

25% 这个数偏低,是因为数据集按设计以不同根因为主(16 份里只有 4 份是重复)。
真实历史里同一失败会反复出现很多次,碰撞率会高得多 —— 而本实验要证明的不是
「数字大」,是上面两条:**该撞的撞了,不该撞的没撞。** 这正是知识库有没有用的判据。

## 规则版本与历史对齐

指纹含规则版本号(当前 `RuleVersion = 1`)。规则一升级:

- 新旧指纹天然不混淆(版本号进了哈希)。
- `cmd/tally reindex-fingerprints` 用留存的原始日志(`failure_event.raw_log_ref`)
  重算历史指纹,报告哪些漂移、哪些没变。事实面只追加(INV-5),该命令只读不写;
  落回可重建的索引面是 reindex(T08)的职责。

## 局限

- 样本量小(16 份),是形状验证不是规模压测。上真实历史数据后应重跑本实验。
- 碰撞率只统计精确指纹;模糊指纹的召回效果要在 T06 诊断流程里评估。
- 还没做「修复后复发」的纵向对齐(需要真实时间线数据)。
