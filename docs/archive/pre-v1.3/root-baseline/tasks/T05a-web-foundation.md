# T05a Web 地基 + 项目管理 CRUD

## 前置
T00(需要 store、jobs、登录中间件、架构测试就位)

## 目标
把平台骨架立起来:能登录、能增改项目/需求/任务/文档/Wiki 笔记。**这是第一个能用的界面里程碑。**

## 交付物

> 登录/CSRF/session 中间件与 `migrations/0009_session.sql` 由 **T00** 交付;本任务在其上建页面。

```
internal/web/
  render.go         html/template + go:embed + 指纹条纯函数
  handlers/
    projects.go     项目列表 / 详情 / 增改(写 git projects/)
    requirements.go 需求条款列表 / 编辑(写 git .tally/requirements/)
    tasks.go        任务列表(状态算出,只读展示)
    docs.go         文档渲染 / 编辑(写 git markdown)
    wiki.go         Wiki 笔记层:列表 / 编辑(写 git wiki/)
web/templates/      base + 各页模板(由 web/prototype/ 转)
web/static/         tally.css(照 06-ui-spec 色板)+ htmx.min.js
```

## 验收标准

### 登录与写边界
- [ ] 未登录访问任何非公开路由 → 重定向 `/login`(INV-9,`auth_required_test.go` 通过)
- [ ] 所有非 GET/HEAD 路由带 CSRF 校验,缺 token 拒绝
- [ ] 密码用 `x/crypto/bcrypt` 哈希;session 存表;`go.mod` 无新依赖
- [ ] **没有任何写路由改状态字段或事实面表**(INV-3/INV-5,架构测试通过)

### 项目管理 CRUD
- [ ] 项目增改 → 落 `projects/` git 并 commit,可 diff/回滚
- [ ] 需求条款编辑 → 落 `.tally/requirements/` git
- [ ] 任务列表状态**从执行链算出展示,页面上没有勾选框/手动状态控件**
- [ ] 文档/Wiki 笔记编辑 → 落对应 git markdown,标「笔记·未验证」

### 质量下限
- [ ] 关掉 JS,表单退化为整页提交仍可用
- [ ] 指纹条是纯函数(同指纹处处同色块)
- [ ] 空状态是行动提示,不是「暂无数据」
- [ ] 窄屏可用、键盘焦点可见、`prefers-reduced-motion` 生效

## 不要做
- 不做多用户/账号体系(单用户)
- 不引入 npm、打包器、CSS 框架
- 不做「标记任务完成」按钮(状态算出)
- 不在此任务接环境/部署/失败数据(那是 T05b)

## 相关文档
`docs/06-ui-spec.md` · `CLAUDE.md`「界面的写边界」· `web/prototype/`
