# T05b 执行引擎 UI + 动作层

## 前置
T01 T03 T04(数据就位);T05a(Web 地基、登录、模板)

## 目标
把环境、部署、失败接上界面,并提供动作按钮。**动作一律入队 jobs,界面不直接改事实面。**

## 交付物

```
internal/web/handlers/
  environments.go   环境状态 + 一键拉起/推倒/重建(入队)
  deploys.go        部署历史 + 一键回滚(入队)
  failures.go       失败列表(按指纹归并)+ 修复排行 + 时间线
  fingerprint.go    指纹详情(视图 B 的执行侧:影响项目/已验证修复/漂移)
  actions.go        动作入队 helper(统一写 jobs)
web/templates/      对应页面模板
```

## 验收标准

### 动作边界(核心)
- [ ] 「拉起/推倒/重建/回滚」POST 后**只入队 job**,返回「已入队」;结果在刷新时从事实面读出
- [ ] **没有任何 handler 直接 INSERT/UPDATE `environment_run`/`deploy_run`**(那是 worker 的事,INV-5)
- [ ] 破坏性动作(推倒重建、回滚)用警示色 + 二次确认,文案说清后果
- [ ] 所有写路由过登录 + CSRF(INV-9)

### 展示
- [ ] 失败按指纹归并,指纹条同指纹处处一致(纯函数)
- [ ] 修复排行 = 出现次数 × 影响项目数
- [ ] 失败时间线与 `dependency_change` 对齐显示
- [ ] 空状态是行动提示;关 JS 动作按钮退化为整页提交仍可用

## 不要做
- 不实现部署/编排逻辑(那是 T03 外包给 Kamal/Ansible 的)
- 不做实时日志流/构建日志查看器(CI 已有)
- 不在此任务接需求维度(视图 B 的关联条款在 T05c 用 LEFT JOIN 补)

## 相关文档
`docs/06-ui-spec.md` ②③ · `CLAUDE.md`「界面的写边界」
