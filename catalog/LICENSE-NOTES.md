# 许可证笔记(抽取开工前的硬节点)

> 本文件是 T02 抽取的**开工前置**:没有它不允许开始抽取(PRD 第一部分规则 ③)。
> 结论来自 2026-07-22 对 Coolify 仓库的**实际查证**(GitHub 仓库 / API / 原始文件),
> 查证对象为默认分支 `v4.x`。**这是法律事项,宁可保守:查不清的一律标「需人工确认」,绝不臆断。**

---

## 一、查到的事实(均经 curl 实查,附来源)

### 1. Coolify 主项目许可 = Apache License 2.0

- 仓库根有 `LICENSE` 文件,内容为 **Apache License, Version 2.0**(全文,非引用)。
- GitHub 仓库元数据 `license` 字段:`"key":"apache-2.0"` / `"spdx_id":"Apache-2.0"`。
- 来源:`raw.githubusercontent.com/coollabsio/coolify/main/LICENSE`、`api.github.com/repos/coollabsio/coolify`。

**Apache-2.0 是宽松型许可**:允许复制、分发、修改、制作衍生作品,**不要求开源修改后的代码**(与 GPL 不同)。对我们的「抽出来、改一遍、自己存」做法友好。

### 2. 服务模板目录没有单独的许可声明

- 服务定义在 `templates/compose/*.yaml`,**共 359~362 个**(这也正是「不要批量抽 200+」的原因——全量有三百多个)。
- `templates/` 目录下**没有** `LICENSE` / `LICENSE.md` / `README.md`(逐个探测均 HTTP 404)。
- 仓库根**没有** `NOTICE` 文件(HTTP 404)。
- 结论:模板目录没有单独许可,**继承整个仓库的 Apache-2.0**。

### 3. 单个 compose 文件:无版权/许可/署名头,但带元数据注释

实查 `minio.yaml` / `elasticsearch.yaml` / `vaultwarden.yaml` 等,文件开头是**元数据注释块**,
**没有任何 license / copyright / attribution 头**:

```yaml
# documentation: <上游项目文档 URL>
# slogan: ...
# category: ...
# tags: ...
# logo: svgs/xxx.svg
# port: ...
services:
  ...
  image: quay.io/minio/minio:latest   # ← 指向上游镜像
```

- 即:**单个文件层面没有额外署名要求需要保留**(没有 NOTICE、没有版权头)。
- 但每个文件都**引用了上游 Docker 镜像**(见第 5 点,那是另一套许可)。

### 4. 贡献者协议 = 确认清单,不是版权转让 CLA

- `CONTRIBUTING.md` 与 PR 模板里的「Contributor Agreement」是一个**勾选清单**(已读指南、已查重、已在本地实例测过),
  **不是**单独的版权转让/独占许可协议。
- 来源:`raw.githubusercontent.com/coollabsio/coolify/v4.x/.github/pull_request_template.md`、`CONTRIBUTING.md`。
- 含义:社区贡献的模板是**按 Apache-2.0 提交进来的**(Apache-2.0 第 5 条:未另行声明者,提交即按本许可授权,inbound = outbound)。
  这降低了「某个贡献者没授权」的风险——但**不能 100% 排除个别文件抄自他处、带着别的许可**(见需人工确认 ①)。

---

## 二、可以做 / 怎么做

### 可以做的

- **抽取并复制**这些 compose 定义:Apache-2.0 明确允许复制与制作衍生作品。
- **修改**它们(LLM 重写 + 我们验证,本来就是改):允许。
- **存进自己的 git**(自用、自部署):私人使用本就无争议;即便分发也在 Apache-2.0 允许范围内。
- **抽完即脱离 Coolify 运行时**:我们只取公开的定义文件,不取它的代码、不跑它的运行时——这与「不 fork、不依赖运行时」(铁律 6)一致,许可上也站得住。

### 必须履行的义务(Apache-2.0 第 4 条)

1. **随附一份许可副本/指引**(4a):本文件即承担此角色,并注明出处
   —— `https://github.com/coollabsio/coolify`(Apache-2.0),抽取分支 `v4.x`。
2. **标注「已修改」**(4b):每条入库定义在 `verified.json` 里记录来源(`source`/`ref`)与「经 LLM 改写 + 本地验证」,
   表明它不是 Coolify 原样副本。
3. **保留文件内的归属/版权声明**(4c):经实查,Coolify 无 NOTICE 文件、compose 文件无版权头,
   **目前没有需要逐字保留的归属文本**。若抽取时某个文件出现版权头,则原样保留(见需人工确认 ①)。

### 署名怎么做(建议,非强制)

- 每条入库定义保留一个 `source` 字段,指回 Coolify 模板(如 `coolify:minio`)+ 上游文档 URL。
- 本文件 + 每条 `verified.json` 的出处字段构成归属链。这已满足 Apache-2.0(它本身不强制显式署名,只要求保留已有声明与许可副本)。

### 商标(Apache-2.0 第 6 条:不授予商标权)

- **不得**用 Coolify 的名称 / logo 暗示官方背书或关联。
- 把「Coolify」作为**事实性的来源说明**(「本定义改编自 Coolify 服务模板」)是允许的、也是推荐的做法。

---

## 三、要排除 / 要人工核的

### 需人工确认 ① —— 单个文件的来源是否混杂

Coolify 模板是社区贡献,**无法仅凭仓库结构保证每个文件都干净地处于 Apache-2.0 之下**。
个别文件可能抄自上游项目、带着**不同的许可**(例如某上游是 GPL / AGPL,抄进来的 compose 若构成其衍生,可能附带义务)。

- **做法**:每抽一个服务,**人工核对该文件头部有无许可/版权声明**;若有非 Apache-2.0 的声明,
  按那个许可处理或**直接排除该服务**。
- 实查的 `minio` / `elasticsearch` / `vaultwarden` 等**均无**此类头部,但**这不代表全部 359 个都干净**——逐条核是抽取流程的一部分。

### 需人工确认 ② —— 被部署软件(上游镜像)自己的许可

**compose 文件是 Apache-2.0,但它拉起的镜像各有自己的许可,两件事分开算。** 抽定义 = 拿到一份编排配置,
不等于获得所部署软件的授权。示例(均为公开信息,使用前仍需自行核对当前条款):

| 软件 | 镜像 | 许可注意 |
|---|---|---|
| MinIO | quay.io/minio/minio | **AGPL-3.0**(自部署自用通常无碍;若对外提供服务需注意 AGPL 义务) |
| Elasticsearch | docker.elastic.co/... | **Elastic License / SSPL 系**,**非 OSI 开源**;自托管内部使用允许,转售/托管服务受限 |
| Redis | redis | 近年改为**双许可(RSALv2 + SSPLv1)**;要纯开源可考虑 Valkey 替代 |
| Keycloak / Grafana / n8n / Plausible / Umami / Vaultwarden | 各自 | 多为 AGPL / Apache-2.0 / 宽松许可,**逐条核对** |

- **做法**:抽取清单里每个服务,**人工确认目标软件的许可与「自部署个人使用」场景相容**;
  不相容或拿不准的,排除或替换(如 Redis → Valkey)。
- 自用、自部署、不分发的场景下,上述大多可用;**但 AGPL 对外提供、Elastic/Redis 的托管限制等是真实边界,需人确认**。

### 需人工确认 ③ —— 抽取时点的许可状态

- 许可可能随版本变化。本笔记查的是 `v4.x`(2026-07-22)。**正式抽取那一刻,复核一次**仓库 LICENSE 与模板目录有无新增 NOTICE / 单文件许可。
- `service-templates.json` 里的文档链接带 `?utm_source=coolify.io` 跟踪参数——无许可影响,入库时建议去掉跟踪参数。

---

## 四、一句话结论

Coolify 主项目与服务模板均为 **Apache-2.0**,无 NOTICE、无单文件署名头——
**抽取、改写、自存是可以的**,义务是:**随附许可指引(本文件)、标注已修改、保留可能出现的版权声明、不用其商标暗示背书**。
两处必须人工逐条核:**①单个模板来源是否混杂(有无别的许可头);②被部署软件自己的许可(AGPL / Elastic / Redis 等)与自用场景是否相容。**
拿不准的**排除或替换**,不臆断。
